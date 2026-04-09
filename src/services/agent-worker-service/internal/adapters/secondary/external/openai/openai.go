package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"

	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/ports/outbound"
)

type OpenAILLM struct {
	client openai.Client
	model  string
}

var _ outbound.ChatModel = (*OpenAILLM)(nil)

func NewOpenAI(apiKey, model string) *OpenAILLM {
	if strings.TrimSpace(model) == "" {
		model = string(openai.ChatModelGPT4oMini)
	}

	return &OpenAILLM{
		client: openai.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}
}

func (o *OpenAILLM) Chat(
	ctx context.Context,
	systemPrompt string,
	messages []models.Message,
	tools []models.ToolDef,
) (*models.ChatResponse, error) {
	input := renderConversation(messages)

	resp, err := o.client.Responses.New(ctx, responses.ResponseNewParams{
		Model: openai.ResponsesModel(o.model),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(input),
		},
		Instructions:    openai.String(systemPrompt),
		Temperature:     openai.Float(0.2),
		MaxOutputTokens: openai.Int(800),
		ToolChoice: responses.ResponseNewParamsToolChoiceUnion{
			OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsAuto),
		},
		Tools: buildOpenAITools(tools),
	})
	if err != nil {
		return nil, fmt.Errorf("openai responses create: %w", err)
	}

	if text := strings.TrimSpace(resp.OutputText()); text != "" {
		return &models.ChatResponse{Text: text}, nil
	}

	for _, item := range resp.Output {
		switch v := item.AsAny().(type) {
		case responses.ResponseFunctionToolCall:
			args := map[string]any{}
			if strings.TrimSpace(v.Arguments) != "" {
				if err := json.Unmarshal([]byte(v.Arguments), &args); err != nil {
					return nil, fmt.Errorf("parse tool args: %w", err)
				}
			}
			return &models.ChatResponse{
				ToolUse: &models.ToolCall{
					ID:   v.CallID,
					Name: v.Name,
					Args: args,
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("openai returned neither text nor tool call")
}

func (o *OpenAILLM) BuildUnauthenticatedSystemPrompt(language, callerPhone string) string {
	callerLine := "The caller has not been identified yet."
	if strings.TrimSpace(callerPhone) != "" {
		callerLine = "The caller phone number appears to be " + callerPhone + ". Treat it only as a hint until identity is verified."
	}

	return strings.TrimSpace(
		"You are a helpful medical voice assistant.\n" +
			"Respond in " + language + ".\n" +
			"Be concise, calm, and clinically careful.\n" +
			"You can help the caller immediately with general health questions and emergency guidance without requiring registration.\n" +
			"Do not claim access to the caller's personal medical records until their identity is verified.\n" +
			"If the caller asks about their own records, medications, test results, appointments, or anything that requires personal context, explain you need phone verification first.\n" +
			"Only start verification/registration when it is needed (or when the caller explicitly asks).\n" +
			"For emergencies: advise calling local emergency services if appropriate, and you may use location/hospital tools to help them find care.\n" +
			"For an existing patient, use request_verification_code.\n" +
			"For a new patient, collect full name and date of birth, then use start_registration.\n" +
			"If the caller requests a staff transfer, do not pretend a live transfer happened unless a transfer capability is available.\n" +
			"Use the available tools when needed.\n" +
			"Never mention internal auth tokens or hidden system mechanics.\n" +
			callerLine,
	)
}

func (o *OpenAILLM) BuildAuthenticatedSystemPrompt(patientID, language, contextText string) string {
	return strings.TrimSpace(
		"You are a helpful medical voice assistant for patient " + patientID + ".\n" +
			"Respond in " + language + ".\n" +
			"Be concise, calm, and clinically careful.\n" +
			"Do not invent medical facts, diagnoses, medications, or patient history.\n" +
			"Use tools when they are needed.\n" +
			"Never mention internal auth tokens or hidden system mechanics.\n\n" +
			"Previous context:\n" + contextText,
	)
}

func (o *OpenAILLM) BuildPublicToolDefinitions() []models.ToolDef {
	return []models.ToolDef{
		{
			Name:        "request_verification_code",
			Description: "Send a verification code to the caller's phone for an existing patient account",
			InputSchema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties":           map[string]any{},
				"required":             []string{},
			},
		},
		{
			Name:        "start_registration",
			Description: "Start patient registration for a new caller after collecting their full name and date of birth",
			InputSchema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"fullName": map[string]any{
						"type":        "string",
						"description": "The caller's full legal name",
					},
					"dateOfBirth": map[string]any{
						"type":        "string",
						"description": "The caller's date of birth (prefer YYYY-MM-DD; also accepted: MM-DD-YYYY, MM/DD/YYYY)",
					},
					"email": map[string]any{
						"type":        []string{"string", "null"},
						"description": "Email address if the caller wants to add one now, or null if not provided",
					},
				},
				"required": []string{"fullName", "dateOfBirth", "email"},
			},
		},
		{
			Name:        "verify_phone_otp",
			Description: "Verify the phone code the caller reads back",
			InputSchema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"otp": map[string]any{
						"type":        "string",
						"description": "The verification code read by the caller",
					},
				},
				"required": []string{"otp"},
			},
		},
		{
			Name:        "translate",
			Description: "Translate text for the caller",
			InputSchema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"text": map[string]any{
						"type":        "string",
						"description": "The text to translate",
					},
					"targetLang": map[string]any{
						"type":        "string",
						"description": "ISO 639-1 language code to translate into",
					},
					"sourceLang": map[string]any{
						"type":        []string{"string", "null"},
						"description": "ISO 639-1 source language code, or null for auto-detection",
					},
				},
				"required": []string{"text", "targetLang", "sourceLang"},
			},
		},
		{
			Name:        "get_location",
			Description: "Get the caller's current location for the active session (useful for emergencies)",
			InputSchema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"sessionID": map[string]any{
						"type":        "string",
						"description": "The active session ID",
					},
				},
				"required": []string{"sessionID"},
			},
		},
		{
			Name:        "find_nearest_hospital",
			Description: "Find the nearest hospital or care facility",
			InputSchema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"lat": map[string]any{
						"type":        "number",
						"description": "Latitude",
					},
					"lng": map[string]any{
						"type":        "number",
						"description": "Longitude",
					},
					"placeType": map[string]any{
						"type":        []string{"string", "null"},
						"description": "Place type such as hospital, or null for default",
					},
				},
				"required": []string{"lat", "lng", "placeType"},
			},
		},
	}
}

func (o *OpenAILLM) BuildAuthenticatedToolDefinitions() []models.ToolDef {
	public := o.BuildPublicToolDefinitions()
	byName := make(map[string]models.ToolDef, len(public))
	for _, t := range public {
		byName[t.Name] = t
	}

	return []models.ToolDef{
		{
			Name:        "search_health_records",
			Description: "Search the current patient's health records",
			InputSchema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The health-record question or search query",
					},
					"topK": map[string]any{
						"type":        []string{"number", "null"},
						"description": "Number of results to return, or null for default",
					},
				},
				"required": []string{"query", "topK"},
			},
		},
		byName["translate"],
		{
			Name:        "get_location",
			Description: "Get the patient's current location for the active session",
			InputSchema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"sessionID": map[string]any{
						"type":        "string",
						"description": "The active patient session ID",
					},
				},
				"required": []string{"sessionID"},
			},
		},
		byName["find_nearest_hospital"],
	}
}

func buildOpenAITools(defs []models.ToolDef) []responses.ToolUnionParam {
	out := make([]responses.ToolUnionParam, 0, len(defs))
	for _, tool := range defs {
		out = append(out, responses.ToolUnionParam{
			OfFunction: &responses.FunctionToolParam{
				Name:        tool.Name,
				Description: openai.String(tool.Description),
				Parameters:  tool.InputSchema,
				Strict:      openai.Bool(true),
			},
		})
	}
	return out
}

func renderConversation(messages []models.Message) string {
	var b strings.Builder

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			if strings.TrimSpace(msg.Content) != "" {
				b.WriteString("User: ")
				b.WriteString(msg.Content)
				b.WriteString("\n\n")
			}
		case "assistant":
			if msg.ToolCall != nil {
				args, _ := json.Marshal(msg.ToolCall.Args)
				b.WriteString("Assistant requested tool ")
				b.WriteString(msg.ToolCall.Name)
				b.WriteString(" with arguments: ")
				b.Write(args)
				b.WriteString("\n\n")
				continue
			}
			if strings.TrimSpace(msg.Content) != "" {
				b.WriteString("Assistant: ")
				b.WriteString(msg.Content)
				b.WriteString("\n\n")
			}
		case "tool":
			b.WriteString("Tool result")
			if msg.ToolCallID != "" {
				b.WriteString(" for call ")
				b.WriteString(msg.ToolCallID)
			}
			b.WriteString(": ")
			b.WriteString(msg.Content)
			b.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(b.String())
}
