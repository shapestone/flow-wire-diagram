# Write Flow Diagram

A complex real-world diagram with two outer boxes, nested inner boxes, connectors,
and content lines where the outer right wall is one column too far right on many lines.

```ascii
┌─────────────────────────────────────────────────────────┐
│                     FRONTEND                             │
│                                                          │
│  WriteChat.vue                                           │
│  ┌─────────────┐    emit('send-message')                │
│  │ Chat Input  │─────────────────────────┐              │
│  └─────────────┘                         ▼              │
│                                   WriteView.vue          │
│                                   (quality enrichment)   │
│                                          │              │
│                                          ▼              │
│                                  useWriteSession.ts      │
│                                  ┌──────────────────┐   │
│                                  │ 1. Add user msg  │   │
│                                  │ 2. Set streaming  │   │
│  ┌───── chunks ◄─────────────── │ 3. WS send       │   │
│  │  streamingText += chunk       └──────────────────┘   │
│  │                                        │ WebSocket   │
│  ▼                                        ▼             │
│  WriteChat.vue ◄── write:done ── useWebSocket.ts        │
│  (render markdown)                                       │
└─────────────────────────────────────────────────────────┘
                            │ ws://localhost:8080/api/v1/ws
                            ▼
┌─────────────────────────────────────────────────────────┐
│                      BACKEND                             │
│                                                          │
│  websocket.go                                            │
│  ┌──────────┐    route by type    ┌──────────────────┐  │
│  │ readPump │───────────────────► │ handleWriteMsg() │  │
│  └──────────┘                     └────────┬─────────┘  │
│                                            │             │
│                                            ▼             │
│  ┌─────────────────────────────────────────────────┐    │
│  │  1. Load session from DB                         │    │
│  │  2. Detect sub-agent (keyword matching)          │    │
│  │  3. Save user message to writing_messages        │    │
│  │  4. Get LLM client (provider + API key from DB)  │    │
│  └──────────────────────┬──────────────────────────┘    │
│                          ▼                               │
│  ┌─────────────────────────────────────────────────┐    │
│  │  BUILD SYSTEM PROMPT (24 fragments)              │    │
│  │  ┌────────────────────────────────────────────┐  │    │
│  │  │ Core: personality, role definition         │  │    │
│  │  │ Context: target, action, tone, draft       │  │    │
│  │  │ Voice: fingerprint, patterns, examples     │  │    │
│  │  │ Guidance: target-specific, AI authorship   │  │    │
│  │  │ Format: response fragment templates        │  │    │
│  │  │ Rules: important notes, constraints        │  │    │
│  │  │ Mode: sub-agent prompt addition            │  │    │
│  │  └────────────────────────────────────────────┘  │    │
│  │  BUILD USER PROMPT                               │    │
│  │  ┌────────────────────────────────────────────┐  │    │
│  │  │ Your new message only (no history)         │  │    │
│  │  └────────────────────────────────────────────┘  │    │
│  └──────────────────────┬──────────────────────────┘    │
│                          ▼                               │
│  ┌─────────────────────────────────────────────────┐    │
│  │  ANTHROPIC API (streaming)                       │    │
│  │                                                   │    │
│  │  GenerateStream(model, system, prompt, ...)       │    │
│  │       │                                           │    │
│  │       ├─► chunk → write:chunk → frontend          │    │
│  │       ├─► chunk → write:chunk → frontend          │    │
│  │       ├─► chunk → write:chunk → frontend          │    │
│  │       └─► done                                    │    │
│  └──────────────────────┬──────────────────────────┘    │
│                          ▼                               │
│  ┌─────────────────────────────────────────────────┐    │
│  │  POST-PROCESSING                                 │    │
│  │  1. Extract <suggested_draft> if present          │    │
│  │  2. Save assistant message to writing_messages    │    │
│  │  3. Save draft version (if suggestion)            │    │
│  │  4. Update session timestamp                      │    │
│  │  5. Send write:done with full session             │    │
│  │  6. (Editor only) Async quality check goroutine   │    │
│  └─────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
```
