# Voice Call Flow - Current Agentic Workflows

This document reflects the current voice-agent architecture after the workflow implementation pass. The voice agent now relies on MCP-backed tools for patient verification, emergency escalation, and grounded health-record answers, while hospital staff use the web app for record summaries.

## Patient Voice Session

```mermaid
sequenceDiagram
    participant Caller as Phone Caller
    participant LiveKit as LiveKit
    participant VoiceAgent as Voice Agent Service
    participant STT as Deepgram STT
    participant LLM as OpenAI LLM
    participant TTS as Deepgram/ElevenLabs TTS
    participant MCP as MCP Server
    participant Patient as Patient Service
    participant Records as Health Records Service
    participant Notify as Notification Service
    participant MQ as RabbitMQ
    participant Loc as Location Service
    participant App as Patient app (optional)

    Caller->>LiveKit: SIP phone call joins room
    LiveKit->>VoiceAgent: RTC session starts
    VoiceAgent->>VoiceAgent: Build provisional session token
    VoiceAgent->>VoiceAgent: Read caller phone + preferred language from SIP identity/metadata
    VoiceAgent->>LLM: Start instructions with language + safety guardrails

    loop Each caller turn
        Caller->>LiveKit: Audio
        LiveKit->>STT: Streaming audio
        STT-->>VoiceAgent: Final transcript + detected language
        VoiceAgent->>VoiceAgent: Update session language if needed

        alt Emergency language detected
            VoiceAgent->>MCP: log_escalation
            MCP->>MQ: publish emergency escalation event
            MQ->>Notify: consume escalation event
            Notify->>Notify: send on-call SMS alerts
            VoiceAgent->>LLM: Emergency-only instructions
            LLM-->>VoiceAgent: Brief urgent guidance
        else Needs identity verification
            VoiceAgent->>MCP: lookup/start OTP/complete registration tools
            MCP->>Patient: gRPC verification or registration call
            Patient-->>MCP: patient verified or registration advanced
            MCP-->>VoiceAgent: tool result
            VoiceAgent->>VoiceAgent: mint patient JWT with records:read
            VoiceAgent->>MCP: notify_call_lifecycle call.started
            MCP->>MQ: publish call.started
            MQ->>Loc: consume call.started
            Loc->>App: WS start_location (if app connected + LOCATION_ACCESS)
            App->>Loc: WS location_update (session-linked GPS)
            LLM-->>VoiceAgent: Continue in caller's language
        else Verified patient needs location
            VoiceAgent->>MCP: get_location
            MCP->>Loc: gRPC GetLocation (Redis GPS or IP fallback)
            Loc-->>MCP: lat/lng
            MCP-->>VoiceAgent: location payload
        else Verified patient asks about records
            VoiceAgent->>MCP: answer_health_question
            MCP->>Records: AnswerPatientQuestion gRPC
            Records->>Records: vector search top matching chunks
            Records->>LLM: grounded answer prompt with retrieved chunks only
            LLM-->>Records: concise answer
            Records-->>MCP: answer + citations
            MCP-->>VoiceAgent: answer payload
            LLM-->>VoiceAgent: natural spoken reply
        else General question
            VoiceAgent->>LLM: answer directly
            LLM-->>VoiceAgent: spoken response
        end

        VoiceAgent->>TTS: synthesize reply in session language
        TTS-->>LiveKit: audio
        LiveKit-->>Caller: spoken response
    end
```

## Staff Summary Workflow

```mermaid
sequenceDiagram
    participant Staff as Hospital Staff
    participant Web as Next.js Web App
    participant Gateway as API Gateway
    participant Auth as Auth Service
    participant Records as Health Records Service

    Staff->>Web: Sign in with hospital email/password
    Web->>Gateway: POST /api/v1/auth/hospital/login
    Gateway->>Auth: Login gRPC
    Auth-->>Gateway: staff JWT
    Gateway-->>Web: access token

    Staff->>Web: Submit patient ID + summary focus
    Web->>Gateway: POST /api/v1/hospital/records/summary
    Gateway->>Gateway: verify staff JWT and hospital claims
    Gateway->>Records: SummarizeRecords gRPC with forwarded token
    Records-->>Gateway: summary
    Gateway-->>Web: summary text
    Web-->>Staff: render patient summary
```

## Notes

- Multilingual behavior starts from SIP participant metadata when present and falls back to transcript language detection.
- Emergency handling is per final transcript turn, not just room setup time.
- Record access requires verified patient identity and a scoped patient JWT.
- Grounded patient Q&A and staff summaries are separate flows with separate actor permissions.
