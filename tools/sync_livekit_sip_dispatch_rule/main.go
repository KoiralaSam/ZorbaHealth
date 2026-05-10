package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	lkauth "github.com/livekit/protocol/auth"
	lklivekit "github.com/livekit/protocol/livekit"
)

// Usage:
//   go run ./tools/sync_livekit_sip_dispatch_rule \
//     -livekit-url http://34.198.147.82:7880 \
//     -api-key ... -api-secret ... \
//     -webhook-url https://<your-tunnel>/webhook/livekit \
//     -room-prefix zorba-call- \
//     -rule-name zorba-agent-individual
//
// This will delete any existing SIP dispatch rule whose attributes["agent_webhook_url"]
// matches -webhook-url, then create a new Individual dispatch rule.
func main() {
	var (
		livekitURL = flag.String("livekit-url", getenv("LIVEKIT_URL", ""), "LiveKit base URL (e.g. http://host:7880 or https://livekit.example.com). If empty, will derive from LIVEKIT_WS_URL.")
		apiKey     = flag.String("api-key", getenv("LIVEKIT_API_KEY", ""), "LiveKit API key")
		apiSecret  = flag.String("api-secret", getenv("LIVEKIT_API_SECRET", ""), "LiveKit API secret")

		webhookURL = flag.String("webhook-url", getenv("AGENT_WEBHOOK_URL", ""), "Your agent worker webhook URL (stored on rule attribute agent_webhook_url)")
		ruleName   = flag.String("rule-name", getenv("SIP_DISPATCH_RULE_NAME", "zorba-agent-individual"), "Human-readable dispatch rule name")
		roomPrefix = flag.String("room-prefix", getenv("SIP_DISPATCH_ROOM_PREFIX", "zorba-call-"), "Room prefix for Individual dispatch rule")

		toNumbersRaw   = flag.String("to-numbers", getenv("SIP_DISPATCH_TO_NUMBERS", ""), "Comma-separated callee/DID numbers this rule matches (optional)")
		fromNumbersRaw = flag.String("from-numbers", getenv("SIP_DISPATCH_FROM_NUMBERS", ""), "Comma-separated caller numbers this rule matches (optional)")
		trunkIDsRaw    = flag.String("trunk-ids", getenv("SIP_DISPATCH_TRUNK_IDS", ""), "Comma-separated SIP trunk IDs this rule matches (optional)")
		hidePhone      = flag.Bool("hide-phone-number", getenvBool("SIP_DISPATCH_HIDE_PHONE", false), "If true, omit phone numbers from participant identity/attrs")
		language       = flag.String("language", getenv("SIP_DISPATCH_LANGUAGE", "en"), "Default language to store in dispatch rule metadata (JSON)")

		// Optional cleanup for legacy rules that don't have the agent_webhook_url attribute yet.
		deleteDirectRoom = flag.String("delete-direct-room", getenv("SIP_DISPATCH_DELETE_DIRECT_ROOM", ""), "If set, delete any Direct dispatch rule whose room_name matches this value (e.g. my-room)")
	)
	flag.Parse()

	if strings.TrimSpace(*livekitURL) == "" {
		derived := deriveHTTPBaseURLFromWS(getenv("LIVEKIT_WS_URL", ""))
		if derived == "" {
			fatalf("-livekit-url (or LIVEKIT_URL) is required (or set LIVEKIT_WS_URL so it can be derived)")
		}
		*livekitURL = derived
	}
	if strings.TrimSpace(*apiKey) == "" {
		fatalf("-api-key (or LIVEKIT_API_KEY) is required")
	}
	if strings.TrimSpace(*apiSecret) == "" {
		fatalf("-api-secret (or LIVEKIT_API_SECRET) is required")
	}
	if strings.TrimSpace(*webhookURL) == "" {
		fatalf("-webhook-url (or AGENT_WEBHOOK_URL) is required")
	}

	toNumbers := splitCSV(*toNumbersRaw)
	fromNumbers := splitCSV(*fromNumbersRaw)
	trunkIDs := splitCSV(*trunkIDsRaw)

	token, err := lkauth.NewAccessToken(*apiKey, *apiSecret).
		SetIdentity("zorba-sip-admin").
		SetSIPGrant(&lkauth.SIPGrant{Admin: true}).
		SetValidFor(10 * time.Minute).
		ToJWT()
	if err != nil {
		fatalf("mint LiveKit API JWT: %v", err)
	}

	httpClient := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &authHeaderRoundTripper{
			base:  http.DefaultTransport,
			token: token,
		},
	}
	sip := lklivekit.NewSIPProtobufClient(*livekitURL, httpClient)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1) List all dispatch rules and delete matches.
	var deletedIDs []string
	var afterID string
	const pageSize = 200
	for {
		resp, err := sip.ListSIPDispatchRule(ctx, &lklivekit.ListSIPDispatchRuleRequest{
			Page: &lklivekit.Pagination{
				AfterId: afterID,
				Limit:   pageSize,
			},
		})
		if err != nil {
			fatalf("list SIP dispatch rules: %v", err)
		}
		items := resp.GetItems()
		if len(items) == 0 {
			break
		}
		for _, it := range items {
			if it == nil {
				continue
			}
			shouldDelete := it.GetAttributes()["agent_webhook_url"] == *webhookURL
			if !shouldDelete && strings.TrimSpace(*deleteDirectRoom) != "" {
				if direct := it.GetRule().GetDispatchRuleDirect(); direct != nil && direct.GetRoomName() == *deleteDirectRoom {
					shouldDelete = true
				}
			}
			if !shouldDelete {
				continue
			}
			id := it.GetSipDispatchRuleId()
			if strings.TrimSpace(id) == "" {
				continue
			}
			if _, err := sip.DeleteSIPDispatchRule(ctx, &lklivekit.DeleteSIPDispatchRuleRequest{
				SipDispatchRuleId: id,
			}); err != nil {
				fatalf("delete SIP dispatch rule %q: %v", id, err)
			}
			deletedIDs = append(deletedIDs, id)
		}

		afterID = items[len(items)-1].GetSipDispatchRuleId()
		if len(items) < pageSize {
			break
		}
	}

	// 2) Create the new Individual dispatch rule.
	created, err := sip.CreateSIPDispatchRule(ctx, &lklivekit.CreateSIPDispatchRuleRequest{
		DispatchRule: &lklivekit.SIPDispatchRuleInfo{
			Rule: &lklivekit.SIPDispatchRule{
				Rule: &lklivekit.SIPDispatchRule_DispatchRuleIndividual{
					DispatchRuleIndividual: &lklivekit.SIPDispatchRuleIndividual{
						RoomPrefix: *roomPrefix,
					},
				},
			},
			Name:            *ruleName,
			TrunkIds:        trunkIDs,
			Numbers:         toNumbers,
			InboundNumbers:  fromNumbers,
			HidePhoneNumber: *hidePhone,
			Metadata:        fmt.Sprintf("{\"language\":%q}", *language),
			Attributes: map[string]string{
				"agent_webhook_url": *webhookURL,
			},
		},
	})
	if err != nil {
		fatalf("create SIP dispatch rule: %v", err)
	}

	fmt.Printf("deleted_dispatch_rule_ids=%s\n", strings.Join(deletedIDs, ","))
	fmt.Printf("created_dispatch_rule_id=%s\n", created.GetSipDispatchRuleId())
	fmt.Printf("created_rule_type=individual\n")
	fmt.Printf("created_room_prefix=%s\n", created.GetRule().GetDispatchRuleIndividual().GetRoomPrefix())
	fmt.Printf("created_rule_attrs.agent_webhook_url=%s\n", created.GetAttributes()["agent_webhook_url"])
}

type authHeaderRoundTripper struct {
	base  http.RoundTripper
	token string
}

func (t *authHeaderRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	rr := r.Clone(r.Context())
	rr.Header = rr.Header.Clone()
	rr.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(rr)
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func deriveHTTPBaseURLFromWS(wsURL string) string {
	wsURL = strings.TrimSpace(wsURL)
	if wsURL == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(wsURL, "ws://"):
		return "http://" + strings.TrimPrefix(wsURL, "ws://")
	case strings.HasPrefix(wsURL, "wss://"):
		return "https://" + strings.TrimPrefix(wsURL, "wss://")
	default:
		// Already looks like an HTTP base URL or something else; accept as-is.
		if strings.HasPrefix(wsURL, "http://") || strings.HasPrefix(wsURL, "https://") {
			return wsURL
		}
		return ""
	}
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
