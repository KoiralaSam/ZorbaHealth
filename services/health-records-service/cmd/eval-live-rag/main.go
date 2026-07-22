// Live RAG harness: ingest curated FHIR gold patients into local Postgres,
// answer gold QA with real OpenAI embeddings/chat, write metrics JSON.
//
// Must live under services/health-records-service/ so it can import internal/.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	openaiadapter "github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/adapters/secondary/openai"
	postgresrepo "github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/adapters/secondary/repositories/postgres"
	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/fhir"
	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/rag"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
)

type goldRow struct {
	ID                     string `json:"id"`
	PatientID              string `json:"patient_id"`
	BundleFile             string `json:"bundle_file"`
	Question               string `json:"question"`
	ExpectedAnswerContains any    `json:"expected_answer_contains"`
	ExpectedBehavior       string `json:"expected_behavior"`
	QuestionType           string `json:"question_type"`
	MustCite               bool   `json:"must_cite"`
	ForbiddenClaims        []string `json:"forbidden_claims"`
}

type itemResult struct {
	ID            string `json:"id"`
	PatientID     string `json:"patient_id"`
	OK            bool   `json:"ok"`
	Behavior      string `json:"behavior"`
	HitAnswer     bool   `json:"hit_answer,omitempty"`
	HitCitation   bool   `json:"hit_citation,omitempty"`
	MustCiteOK    bool   `json:"must_cite_ok"`
	CitationCount int    `json:"citation_count"`
	AnswerPreview string `json:"answer_preview"`
	Error         string `json:"error,omitempty"`
	LatencyMS     int64  `json:"latency_ms"`
}

func main() {
	repoRoot := findRepoRoot()
	goldPath := flag.String("gold", filepath.Join(repoRoot, "examples/evaluation-data/gold/gold_qa.jsonl"), "gold QA jsonl")
	outPath := flag.String("out", filepath.Join(repoRoot, "examples/evaluation-data/gold/live_rag_results.json"), "output json")
	flag.Parse()

	loadDotEnv(filepath.Join(repoRoot, "examples/sample-env/.env.docker"))

	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
	}

	dbURL := env.GetString("DATABASE_URL", "postgres://healthai:healthai@localhost:5432/healthai?sslmode=disable")
	dbURL = strings.Replace(dbURL, "@postgres:", "@localhost:", 1)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()

	store := postgresrepo.NewRepository(pool)
	embedder := openaiadapter.NewClient(apiKey)
	summarizer := openaiadapter.NewSummarizerClient(apiKey)
	ingestor := fhir.NewIngestor(store, embedder, "text-embedding-3-small")
	pipeline := rag.NewPipeline(store, embedder, summarizer, nil, nil, "text-embedding-3-small")

	rows, err := readGold(*goldPath)
	if err != nil {
		log.Fatalf("gold: %v", err)
	}

	internalByFHIR := map[string]uuid.UUID{}
	for _, row := range rows {
		if _, ok := internalByFHIR[row.PatientID]; ok {
			continue
		}
		pid := uuid.New()
		phone := fmt.Sprintf("+1555%07d", len(internalByFHIR)+1)
		_, err := pool.Exec(ctx, `
			INSERT INTO patients (id, phone_number, full_name, email, created_at)
			VALUES ($1, $2, $3, $4, now())`,
			pid, phone, row.PatientID, row.PatientID+"@eval.local")
		if err != nil {
			log.Fatalf("insert patient %s: %v", row.PatientID, err)
		}
		bundlePath := row.BundleFile
		if !filepath.IsAbs(bundlePath) {
			bundlePath = filepath.Join(repoRoot, bundlePath)
		}
		raw, err := os.ReadFile(bundlePath)
		if err != nil {
			log.Fatalf("read bundle %s: %v", bundlePath, err)
		}
		res, err := ingestor.IngestBundle(ctx, pid, string(raw), "eval-gold")
		if err != nil {
			log.Fatalf("ingest %s: %v", row.PatientID, err)
		}
		log.Printf("ingested %s -> %s resources=%d chunks=%d", row.PatientID, pid, res.ResourcesStored, res.ChunksStored)
		internalByFHIR[row.PatientID] = pid
	}

	items := make([]itemResult, 0, len(rows))
	answerable, answerableOK := 0, 0
	unanswerable, unanswerableOK := 0, 0
	citeOK, citeN := 0, 0
	var latencies []int64

	for _, row := range rows {
		pid := internalByFHIR[row.PatientID]
		start := time.Now()
		result, err := pipeline.Run(ctx, rag.QueryRequest{
			PatientID: pid.String(),
			Query:     row.Question,
			TopK:      5,
			Summarize: true,
		})
		elapsed := time.Since(start).Milliseconds()
		latencies = append(latencies, elapsed)

		ir := itemResult{
			ID:        row.ID,
			PatientID: row.PatientID,
			LatencyMS: elapsed,
			Behavior:  row.ExpectedBehavior,
		}
		if err != nil {
			ir.Error = err.Error()
			if row.ExpectedBehavior != "grounded_answer" {
				unanswerable++
				ok := !containsAny(err.Error()+" "+result.Answer, []string{"ldl", "cholesterol"})
				ir.OK = ok
				if ok {
					unanswerableOK++
				}
			} else {
				answerable++
				ir.OK = false
			}
			items = append(items, ir)
			continue
		}

		ir.CitationCount = len(result.Citations)
		ir.AnswerPreview = trim(result.Answer, 240)
		citeText := ""
		for _, c := range result.Citations {
			citeText += c.Text + "\n"
		}
		needles := asStrings(row.ExpectedAnswerContains)

		if row.ExpectedBehavior != "grounded_answer" {
			unanswerable++
			// Safe refusal may mention the lab name while declining; flag numeric value claims.
			ir.OK = isSafeUnanswerable(result.Answer)
			if ir.OK {
				unanswerableOK++
			}
			ir.MustCiteOK = true
			items = append(items, ir)
			continue
		}

		answerable++
		hitAns := containsAny(result.Answer, needles)
		hitCite := containsAny(citeText, needles)
		ir.HitAnswer = hitAns
		ir.HitCitation = hitCite
		ir.OK = hitAns || hitCite
		ir.MustCiteOK = !row.MustCite || len(result.Citations) > 0
		if row.MustCite {
			citeN++
			if ir.MustCiteOK {
				citeOK++
			}
		}
		if ir.OK {
			answerableOK++
		}
		items = append(items, ir)
	}

	summary := map[string]any{
		"mode":                   "live_openai_rag",
		"answerable_n":           answerable,
		"answerable_accuracy":    ratio(answerableOK, answerable),
		"answerable_hits":        answerableOK,
		"unanswerable_n":         unanswerable,
		"unanswerable_safe_rate": ratio(unanswerableOK, unanswerable),
		"citation_presence_rate": ratio(citeOK, citeN),
		"avg_latency_ms":         avg(latencies),
		"p95_latency_ms":         percentile(latencies, 0.95),
		"embedding_model":        "text-embedding-3-small",
		"note":                   "Consent gate bypassed in harness; measures grounded retrieval+answer quality on synthetic gold QA.",
	}
	out := map[string]any{"summary": summary, "items": items}
	b, _ := json.MarshalIndent(out, "", "  ")
	if err := os.WriteFile(*outPath, append(b, '\n'), 0o644); err != nil {
		log.Fatalf("write out: %v", err)
	}
	fmt.Println(string(b))
}

func readGold(path string) ([]goldRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var rows []goldRow
	sc := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r goldRow
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			return nil, err
		}
		rows = append(rows, r)
	}
	return rows, sc.Err()
}

func asStrings(v any) []string {
	switch t := v.(type) {
	case string:
		if t == "" {
			return nil
		}
		return []string{t}
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			if s, ok := x.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return t
	default:
		return nil
	}
}

func containsAny(text string, needles []string) bool {
	low := strings.ToLower(text)
	for _, n := range needles {
		n = strings.TrimSpace(strings.ToLower(n))
		if n != "" && strings.Contains(low, n) {
			return true
		}
	}
	return false
}

func isSafeUnanswerable(answer string) bool {
	low := strings.ToLower(answer)
	refusalHints := []string{
		"do not see", "don't see", "not in", "not found", "not available",
		"no record", "no information", "insufficient", "cannot determine", "unable to",
	}
	hasRefusal := containsAny(low, refusalHints)
	// Numeric LDL/cholesterol fabrication heuristic.
	hasValueClaim := strings.Contains(low, "mg/dl") ||
		(strings.Contains(low, "ldl") && containsDigitNear(low, "ldl", 40))
	return hasRefusal && !hasValueClaim
}

func containsDigitNear(text, needle string, window int) bool {
	idx := strings.Index(text, needle)
	if idx < 0 {
		return false
	}
	end := idx + len(needle) + window
	if end > len(text) {
		end = len(text)
	}
	segment := text[idx:end]
	for _, r := range segment {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func ratio(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}

func avg(xs []int64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var s int64
	for _, x := range xs {
		s += x
	}
	return float64(s) / float64(len(xs))
}

func percentile(xs []int64, p float64) int64 {
	if len(xs) == 0 {
		return 0
	}
	cp := append([]int64(nil), xs...)
	for i := 0; i < len(cp); i++ {
		for j := i + 1; j < len(cp); j++ {
			if cp[j] < cp[i] {
				cp[i], cp[j] = cp[j], cp[i]
			}
		}
	}
	idx := int(p * float64(len(cp)-1))
	return cp[idx]
}

func trim(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func findRepoRoot() string {
	wd, _ := os.Getwd()
	dir := wd
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	return wd
}

func loadDotEnv(path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}
}
