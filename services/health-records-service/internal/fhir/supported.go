package fhir

// SupportedResourceTypes are ingested and indexed for RAG retrieval.
var SupportedResourceTypes = []string{
	"Patient",
	"Practitioner",
	"Organization",
	"Encounter",
	"Observation",
	"Condition",
	"MedicationRequest",
	"AllergyIntolerance",
	"DiagnosticReport",
	"DocumentReference",
	"CarePlan",
}

func isSupportedResourceType(resourceType string) bool {
	for _, t := range SupportedResourceTypes {
		if t == resourceType {
			return true
		}
	}
	return false
}
