package fhir

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// NormalizedResource holds extracted columns for structured queries and chunk provenance.
type NormalizedResource struct {
	ResourceType    string
	ResourceID      string
	DisplayText     string
	ClinicalStatus  string
	EffectiveDate   *time.Time
	SearchableText  string
	FHIRPatientID   string
}

func NormalizeResource(raw json.RawMessage) (NormalizedResource, error) {
	resourceType, resourceID, err := ValidateResourceJSON(raw)
	if err != nil {
		return NormalizedResource{}, err
	}

	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return NormalizedResource{}, err
	}

	out := NormalizedResource{
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}

	switch resourceType {
	case "Patient":
		out.FHIRPatientID = resourceID
		out.DisplayText = joinName(generic)
		out.SearchableText = strings.TrimSpace(out.DisplayText + " " + stringFrom(generic, "gender") + " " + birthDate(generic))
	case "Condition", "AllergyIntolerance":
		out.ClinicalStatus = nestedCodeText(generic, "clinicalStatus")
		out.DisplayText = codeableConceptText(generic, "code")
		out.SearchableText = strings.TrimSpace(out.DisplayText + " " + out.ClinicalStatus)
	case "Observation", "DiagnosticReport":
		out.DisplayText = codeableConceptText(generic, "code")
		out.ClinicalStatus = stringFrom(generic, "status")
		out.EffectiveDate = parseEffective(generic)
		out.SearchableText = strings.TrimSpace(out.DisplayText + " " + valueSummary(generic))
	case "MedicationRequest":
		out.ClinicalStatus = stringFrom(generic, "status")
		out.DisplayText = codeableConceptText(generic, "medicationCodeableConcept")
		out.SearchableText = strings.TrimSpace(out.DisplayText + " " + out.ClinicalStatus)
	default:
		out.DisplayText = codeableConceptText(generic, "code")
		out.ClinicalStatus = stringFrom(generic, "status")
		out.SearchableText = strings.TrimSpace(out.DisplayText + " " + out.ClinicalStatus)
	}

	if out.SearchableText == "" {
		out.SearchableText = resourceType + " " + resourceID
	}
	return out, nil
}

func joinName(m map[string]any) string {
	nameRaw, ok := m["name"].([]any)
	if !ok || len(nameRaw) == 0 {
		return ""
	}
	nameMap, ok := nameRaw[0].(map[string]any)
	if !ok {
		return ""
	}
	given, _ := nameMap["given"].([]any)
	family, _ := nameMap["family"].(string)
	parts := []string{family}
	for _, g := range given {
		if s, ok := g.(string); ok {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " ")
}

func birthDate(m map[string]any) string {
	if v, ok := m["birthDate"].(string); ok {
		return v
	}
	return ""
}

func stringFrom(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func codeableConceptText(m map[string]any, key string) string {
	cc, ok := m[key].(map[string]any)
	if !ok {
		return ""
	}
	if text, ok := cc["text"].(string); ok && text != "" {
		return text
	}
	coding, ok := cc["coding"].([]any)
	if !ok || len(coding) == 0 {
		return ""
	}
	first, ok := coding[0].(map[string]any)
	if !ok {
		return ""
	}
	if display, ok := first["display"].(string); ok {
		return display
	}
	if code, ok := first["code"].(string); ok {
		return code
	}
	return ""
}

func nestedCodeText(m map[string]any, key string) string {
	node, ok := m[key].(map[string]any)
	if !ok {
		return ""
	}
	coding, ok := node["coding"].([]any)
	if !ok || len(coding) == 0 {
		return ""
	}
	first, ok := coding[0].(map[string]any)
	if !ok {
		return ""
	}
	if code, ok := first["code"].(string); ok {
		return code
	}
	return ""
}

func valueSummary(m map[string]any) string {
	if v, ok := m["valueString"].(string); ok {
		return v
	}
	if q, ok := m["valueQuantity"].(map[string]any); ok {
		val, _ := q["value"].(float64)
		unit, _ := q["unit"].(string)
		return strings.TrimSpace(strings.TrimSpace(formatFloat(val)) + " " + unit)
	}
	return ""
}

func formatFloat(v float64) string {
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(v, 'f', 4, 64), "0"), ".")
}

func parseEffective(m map[string]any) *time.Time {
	for _, key := range []string{"effectiveDateTime", "issued", "recordedDate"} {
		if s, ok := m[key].(string); ok && s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return &t
			}
			if t, err := time.Parse("2006-01-02", s); err == nil {
				return &t
			}
		}
	}
	return nil
}
