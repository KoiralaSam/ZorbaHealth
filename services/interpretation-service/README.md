# Interpretation Service

Bridged-call interpretation relay. Translates final STT segments between the
patient and hospital staff during a bridged consult, using per-party
preferences stored in Redis by patient-service.

## Flow

```
voice-agent-service ──POST /internal/interpretation/segment──▶ interpretation-service
                                                                  │  load voice:bridge:{session_id} (Redis)
                                                                  │  pick TARGET party prefs + JWT
                                                                  ▼
                                                          translation-service ──▶ provider (Amazon Translate / llama.cpp)
```

The Redis session contract is shared with patient-service via
`shared/bridge` (`voice:bridge:{sessionID}`, written by
`RequestBridgedCallTransfer` / `ConnectBridgedCall` /
`UpdateBridgedCallTranslation`).

## Endpoint

`POST /internal/interpretation/segment`

```json
{
  "session_id": "RM_xxx",        // bridged session id (LiveKit room SID)
  "participant": "patient",      // who SPOKE this segment: patient | staff
  "text": "hola doctor",
  "source_lang": "es",           // optional ISO 639-1; empty = auto-detect
  "target_hint": "es"            // optional; listener's live-detected language
}
```

`target_hint` is the listening party's language as observed live on the call
(e.g. the SIP-detected patient language). It is used only when the target party
enabled **auto-mode** translation without an explicit `LanguageCode`, letting
the doctor→patient direction interpret instead of falling back to passthrough.

Response:

```json
{
  "session_id": "RM_xxx",
  "target_language": "en",
  "translated_text": "hello doctor",
  "passthrough": false
}
```

### Authentication

When `INTERNAL_SERVICE_SECRET` is set, callers must send it in the
`x-internal-token` header; otherwise the endpoint returns `401`. The relay
also forwards the stored counterparty JWT to translation-service as
`x-forwarded-token`, so translation authorization is enforced per actor.

### Passthrough rules

The **target** (listening) party's preferences gate translation — a segment
spoken by the patient is translated using the **staff** preferences and vice
versa:

1. `passthrough: true` (original text returned) when the target party has
   `Enabled = false` **or** an empty `LanguageCode`. Enabling translation in
   the UI without choosing a language code still results in passthrough.
2. `passthrough: true` when the segment's `source_lang` equals the target
   `LanguageCode` (already in the right language).
3. `language_mode: auto` applies to the **segment source language**: leave
   `source_lang` empty and the translation provider auto-detects it. The
   target language must be explicit (`LanguageCode`) **or** supplied via
   `target_hint` (auto mode only); manual mode still requires `LanguageCode`.
4. `503` when no bridged actor JWT is stored (tokens are refreshed on every
   bridge transfer/connect/preference-update call and cleared on end).
5. `410` when the session status is `ended`; `404` when the session does not
   exist or its TTL expired.

## Environment

| Variable | Default | Purpose |
|----------|---------|---------|
| `INTERPRETATION_SERVICE_HTTP_ADDR` | `:8095` | HTTP listen address |
| `REDIS_ADDR` / `REDIS_PASSWORD` | `redis:6379` / empty | Bridged session store |
| `TRANSLATION_SERVICE_GRPC_ADDR` | `translation-service:50057` | Translation backend |
| `AUDIT_SERVICE_GRPC_ADDR` | `audit-service:50058` | Segment audit events |
| `INTERNAL_SERVICE_SECRET` | empty (open; warning logged) | Caller auth + audit/translation metadata |

## Producer

`voice-agent-service` posts each final patient STT utterance here when
`INTERPRETATION_SERVICE_URL` is configured on the worker, then publishes the
translated segment to the LiveKit room data channel (topic
`zorba.interpretation`) for the hospital web client to render as captions.

## Audit

Every processed segment (translated or passthrough) emits an
`INTERPRETATION_SEGMENT_PROCESSED` audit event with character counts and
language metadata — never the segment text itself.
