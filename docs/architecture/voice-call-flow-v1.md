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
            VoiceAgent->>MCP: lookup/start OTP tools (JWT binds callerPhone + sessionID)
            MCP->>Patient: StartExistingPhoneVerification + voice_session_id
            Patient->>Patient: Redis voice:otp_wait + outbound OTP SMS
            alt DTMF keypad
                Caller->>LiveKit: DTMF digits
                LiveKit->>VoiceAgent: sip_dtmf_received
                VoiceAgent->>MCP: verify_existing_phone_otp (channel=dtmf)
            else Inbound SMS reply
                Caller->>Notify: POST /sms
                Notify->>Patient: ProcessInboundVoiceSms
                Patient->>Patient: Redis voice:verified
                VoiceAgent->>MCP: consume_voice_verification (poll)
            else Spoken or GetDtmfTask
                VoiceAgent->>MCP: verify_existing_phone_otp (channel=spoken)
            end
            MCP->>Patient: gRPC verification
            Patient-->>MCP: patient verified
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

## Bridged Patient-Staff Interpretation

```mermaid
sequenceDiagram
    participant Patient as Patient Phone
    participant PatientApp as Patient Web/Mobile
    participant LiveKit as LiveKit Room
    participant Gateway as API Gateway
    participant PatientSvc as Patient Service
    participant Redis as Redis
    participant Staff as Hospital Staff
    participant StaffWeb as Hospital Web
    participant Agent as Voice Agent
    participant Relay as Interpretation Relay
    participant Translate as Translation Service
    participant AWS as Amazon Translate
    participant Audit as Audit Service

    Patient->>Gateway: POST /patient/calls/bridge-transfer
    Gateway->>PatientSvc: RequestBridgedCallTransfer
    PatientSvc->>Redis: store session + participant prefs
    PatientSvc->>Audit: CALL_TRANSFER_REQUESTED
    Gateway-->>PatientApp: session payload + patient LiveKit token

    Staff->>StaffWeb: Open bridge console
    StaffWeb->>Gateway: GET /hospital/calls/bridge-sessions?status=transfer_requested
    Gateway->>PatientSvc: ListBridgedCallSessions
    PatientSvc->>Redis: scan voice:bridge:* for hospital
    Gateway-->>StaffWeb: pending transfer list
    StaffWeb->>Gateway: POST /hospital/calls/bridge-connect
    Gateway->>PatientSvc: ConnectBridgedCall
    PatientSvc->>Redis: update session status=connected + refresh staff JWT
    PatientSvc->>Audit: CALL_TRANSFER_CONNECTED
    Gateway-->>StaffWeb: session payload + staff LiveKit join token
    StaffWeb->>LiveKit: join room with staff token
    PatientApp->>LiveKit: join room data-only with patient token
    Agent->>LiveKit: detect staff join, enter interpreter mode
    Agent->>LiveKit: unsubscribe SIP patient from raw staff audio track

    loop Preference changes
        Patient->>Gateway: PUT /patient/calls/bridge-translation
        StaffWeb->>Gateway: PUT /hospital/calls/bridge-translation
        Gateway->>PatientSvc: UpdateBridgedCallTranslation
        PatientSvc->>Redis: store per-party prefs
        PatientSvc->>Audit: INTERPRETATION_PREFERENCES_UPDATED
    end

    loop Doctor speaks
        StaffWeb->>LiveKit: publish clinician audio
        LiveKit->>Agent: clinician audio track
        Agent->>Relay: POST /internal/interpretation/segment participant=staff
        Relay->>Redis: resolve patient/staff prefs by session_id
        Relay->>Translate: Translate(text, source_lang, target_lang) with forwarded actor JWT
        Translate->>AWS: TranslateText
        AWS-->>Translate: translated text
        Translate-->>Relay: translated segment
        Relay->>Audit: INTERPRETATION_SEGMENT_PROCESSED
        Agent->>LiveKit: publish TTS in patient language
        Agent->>LiveKit: publish zorba.interpretation caption participant=staff
        LiveKit-->>Patient: hears translated TTS only
        LiveKit-->>StaffWeb: sees staff caption, skips local translated audio playback
        LiveKit-->>PatientApp: receives caption mirror
    end

    loop Patient speaks
        Patient->>LiveKit: raw SIP audio
        LiveKit->>Agent: patient audio
        Note over Relay: voice-agent POSTs /internal/interpretation/segment (x-internal-token)
        Relay->>Redis: resolve patient/staff prefs by session_id
        Relay->>Translate: Translate(text, source_lang, target_lang) with forwarded actor JWT
        Translate->>AWS: TranslateText
        AWS-->>Translate: translated text
        Translate-->>Relay: translated segment
        Relay->>Audit: INTERPRETATION_SEGMENT_PROCESSED
        Agent->>LiveKit: publish zorba.interpretation caption participant=patient
        LiveKit-->>StaffWeb: clinician reads translated caption
        LiveKit-->>PatientApp: receives caption mirror
    end

    StaffWeb->>Gateway: POST /hospital/calls/bridge-end
    Gateway->>PatientSvc: EndBridgedCall
    PatientSvc->>Redis: mark ended
    PatientSvc->>Audit: CALL_BRIDGED_ENDED
```

## Notes

- Multilingual behavior starts from SIP participant metadata when present and falls back to transcript language detection.
- Interpreter mode suppresses the normal conversational LLM flow while the clinician is connected; patient turns are relayed for captions instead of answered by Zorba.
- Staff audio is translated with STT -> translation -> TTS, while patient companion surfaces join the same room data-only for caption/status mirroring.
- Emergency handling is per final transcript turn, not just room setup time.
- Record access requires verified patient identity and a scoped patient JWT.
- Grounded patient Q&A and staff summaries are separate flows with separate actor permissions.
- Bridged patient-staff calls now persist a session-scoped translation model in Redis so patient and staff preferences can be updated independently during a live consult.
- The translation backend is now provider-switched and can use Amazon Translate without the legacy local `translation-model` deployment.
- The interpretation relay requires `x-internal-token` (`INTERNAL_SERVICE_SECRET`) and serves the producer in voice-agent-service (`INTERPRETATION_SERVICE_URL`). Passthrough rules are documented in `services/interpretation-service/README.md`.
- Bridge ops re-stamp the calling actor's JWT in Redis (transfer/connect/preference updates) so long consults keep a fresh token; ending a bridge clears stored JWTs and shortens the key TTL.
- The Redis session schema is shared via `shared/bridge` so patient-service (writer) and interpretation-service (reader) cannot drift.
