package registry

import sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"

type ToolMetadata struct {
	ToolName               string
	Description            string
	RequiredPermissions    []string
	AllowedActorTypes      []string
	RequiresPatientConsent bool
	ConsentType            string
	ReadsPHI               bool
	WritesPHI              bool
	EmergencyAllowed       bool
	AuditRequired          bool
	RateLimitPerMinute     int
	EventType              string
}

var toolMetadata = map[string]ToolMetadata{
	"search_health_records": {
		ToolName:               "search_health_records",
		Description:            "Search the caller's own health records",
		RequiredPermissions:    []string{"records:read"},
		AllowedActorTypes:      []string{"patient"},
		ReadsPHI:               true,
		AuditRequired:          true,
		RateLimitPerMinute:     20,
		EventType:              sharedaudit.EventHealthRecordSearched,
	},
	"answer_health_question": {
		ToolName:               "answer_health_question",
		Description:            "Answer a caller question using their health records",
		RequiredPermissions:    []string{"records:read"},
		AllowedActorTypes:      []string{"patient"},
		ReadsPHI:               true,
		AuditRequired:          true,
		RateLimitPerMinute:     20,
		EventType:              sharedaudit.EventHealthRecordSummarized,
	},
	"summarize_health_records": {
		ToolName:               "summarize_health_records",
		Description:            "Summarize a patient's records",
		RequiredPermissions:    []string{"patient:read"},
		AllowedActorTypes:      []string{"staff"},
		RequiresPatientConsent: true,
		ConsentType:            sharedaudit.ConsentAISummarization,
		ReadsPHI:               true,
		AuditRequired:          true,
		RateLimitPerMinute:     20,
		EventType:              sharedaudit.EventHealthRecordSummarized,
	},
	"summarize_patient_record": {
		ToolName:               "summarize_patient_record",
		Description:            "Legacy alias for summarize_health_records",
		RequiredPermissions:    []string{"patient:read"},
		AllowedActorTypes:      []string{"staff"},
		RequiresPatientConsent: true,
		ConsentType:            sharedaudit.ConsentAISummarization,
		ReadsPHI:               true,
		AuditRequired:          true,
		RateLimitPerMinute:     20,
		EventType:              sharedaudit.EventHealthRecordSummarized,
	},
	"get_patient_profile": {
		ToolName:               "get_patient_profile",
		Description:            "Retrieve structured patient profile resources",
		RequiredPermissions:    []string{"records:read"},
		AllowedActorTypes:      []string{"patient", "staff"},
		RequiresPatientConsent: true,
		ConsentType:            sharedaudit.ConsentHealthRecordAccess,
		ReadsPHI:               true,
		AuditRequired:          true,
		RateLimitPerMinute:     20,
		EventType:              sharedaudit.EventHealthRecordViewed,
	},
	"get_live_location": {
		ToolName:               "get_live_location",
		Description:            "Retrieve the caller's live location",
		RequiredPermissions:    []string{"location:read"},
		AllowedActorTypes:      []string{"patient"},
		RequiresPatientConsent: true,
		ConsentType:            sharedaudit.ConsentLocationAccess,
		AuditRequired:          true,
		RateLimitPerMinute:     30,
		EventType:              sharedaudit.EventLocationRequested,
	},
	"get_location": {
		ToolName:               "get_location",
		Description:            "Legacy alias for get_live_location",
		RequiredPermissions:    []string{"location:read"},
		AllowedActorTypes:      []string{"patient"},
		RequiresPatientConsent: true,
		ConsentType:            sharedaudit.ConsentLocationAccess,
		AuditRequired:          true,
		RateLimitPerMinute:     30,
		EventType:              sharedaudit.EventLocationRequested,
	},
	"find_nearest_hospital": {
		ToolName:               "find_nearest_hospital",
		Description:            "Find nearby emergency care or hospitals",
		AllowedActorTypes:      []string{"patient", "staff"},
		EmergencyAllowed:       true,
		AuditRequired:          true,
		RateLimitPerMinute:     60,
		EventType:              sharedaudit.EventLocationRequested,
	},
	"translate_text": {
		ToolName:               "translate_text",
		Description:            "Translate text for the caller",
		AllowedActorTypes:      []string{"patient", "staff"},
		EmergencyAllowed:       true,
		AuditRequired:          true,
		RateLimitPerMinute:     120,
		EventType:              sharedaudit.EventTranslationRequested,
	},
	"translate": {
		ToolName:               "translate",
		Description:            "Legacy alias for translate_text",
		AllowedActorTypes:      []string{"patient", "staff"},
		EmergencyAllowed:       true,
		AuditRequired:          true,
		RateLimitPerMinute:     120,
		EventType:              sharedaudit.EventTranslationRequested,
	},
	"get_call_summary": {
		ToolName:               "get_call_summary",
		Description:            "Load the current caller summary/context",
		RequiredPermissions:    []string{"records:read"},
		AllowedActorTypes:      []string{"patient"},
		RequiresPatientConsent: true,
		ConsentType:            sharedaudit.ConsentHealthRecordAccess,
		ReadsPHI:               true,
		AuditRequired:          true,
		RateLimitPerMinute:     20,
		EventType:              sharedaudit.EventHealthRecordViewed,
	},
	"log_escalation": {
		ToolName:               "log_escalation",
		Description:            "Record an emergency escalation event",
		AllowedActorTypes:      []string{"patient", "staff"},
		EmergencyAllowed:       true,
		AuditRequired:          true,
		WritesPHI:              true,
		RateLimitPerMinute:     20,
		EventType:              sharedaudit.EventEmergencyEscalationTriggered,
	},
	"get_hospital_analytics": {
		ToolName:               "get_hospital_analytics",
		Description:            "Get hospital analytics summary",
		RequiredPermissions:    []string{"hospital:analytics"},
		AllowedActorTypes:      []string{"staff"},
		AuditRequired:          true,
		RateLimitPerMinute:     20,
		EventType:              sharedaudit.EventAIToolCalled,
	},
}

func Lookup(toolName string) (ToolMetadata, bool) {
	meta, ok := toolMetadata[toolName]
	return meta, ok
}

func All() map[string]ToolMetadata {
	out := make(map[string]ToolMetadata, len(toolMetadata))
	for k, v := range toolMetadata {
		out[k] = v
	}
	return out
}
