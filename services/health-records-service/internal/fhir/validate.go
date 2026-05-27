package fhir

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ValidateResourceJSON ensures the payload is JSON with a supported FHIR resourceType.
func ValidateResourceJSON(raw json.RawMessage) (string, string, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return "", "", fmt.Errorf("invalid json: %w", err)
	}
	rtRaw, ok := envelope["resourceType"]
	if !ok {
		return "", "", fmt.Errorf("missing resourceType")
	}
	var resourceType string
	if err := json.Unmarshal(rtRaw, &resourceType); err != nil {
		return "", "", fmt.Errorf("invalid resourceType: %w", err)
	}
	resourceType = strings.TrimSpace(resourceType)
	if resourceType == "" {
		return "", "", fmt.Errorf("empty resourceType")
	}
	if !isSupportedResourceType(resourceType) {
		return "", "", fmt.Errorf("unsupported resourceType %q", resourceType)
	}

	id := ""
	if idRaw, ok := envelope["id"]; ok {
		_ = json.Unmarshal(idRaw, &id)
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return "", "", fmt.Errorf("missing resource id for %s", resourceType)
	}
	return resourceType, id, nil
}

// ValidateBundleJSON validates a FHIR Bundle and returns entry resource payloads.
func ValidateBundleJSON(bundleJSON string) ([]json.RawMessage, error) {
	var bundle struct {
		ResourceType string            `json:"resourceType"`
		Type         string            `json:"type"`
		Entry        []json.RawMessage `json:"entry"`
	}
	if err := json.Unmarshal([]byte(bundleJSON), &bundle); err != nil {
		return nil, fmt.Errorf("invalid bundle json: %w", err)
	}
	if bundle.ResourceType != "Bundle" {
		return nil, fmt.Errorf("expected Bundle, got %q", bundle.ResourceType)
	}
	if len(bundle.Entry) == 0 {
		return nil, fmt.Errorf("bundle has no entries")
	}

	var resources []json.RawMessage
	for i, entryRaw := range bundle.Entry {
		var entry struct {
			Resource json.RawMessage `json:"resource"`
		}
		if err := json.Unmarshal(entryRaw, &entry); err != nil {
			return nil, fmt.Errorf("entry %d: %w", i, err)
		}
		if len(entry.Resource) == 0 {
			return nil, fmt.Errorf("entry %d: missing resource", i)
		}
		if _, _, err := ValidateResourceJSON(entry.Resource); err != nil {
			return nil, fmt.Errorf("entry %d: %w", i, err)
		}
		resources = append(resources, entry.Resource)
	}
	return resources, nil
}
