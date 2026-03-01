# Cross-Channel Command Single Source and Platform Registration Design

## Background

Telegram commands are currently defined in both `pkg/channels/telegram/telegram.go` and `pkg/channels/telegram/telegram_commands.go`, which causes the following issues:

- Adding a new command requires duplicated updates in multiple places, which is error-prone.
- Telegram platform menu commands (Bot Commands) are not automatically registered.
- When expanding to channels such as WhatsApp, command definitions and behavior cannot be reused.

## Goals

- Establish a single source of truth for command definitions.
- When adding a new command, update one definition and automatically synchronize it to:
  - command parsing and execution;
  - Telegram platform command menu registration;
  - `help` command output.
- Support layered multi-channel capabilities:
  - all channels share one command parsing/execution flow;
  - platform registration is enabled only for channels that support registration APIs (for example, Telegram).
- Command registration failures at startup must not block channel startup; use warnings plus background retries.

## Non-Goals

- This iteration does not require all channels to provide platform-side command menus.
- Do not change the Agent's main business-message handling logic; only command entry and registration layers are in scope.
- Do not introduce a complex command permission system (YAGNI; reuse the existing allow-list mechanism).

## Decision Log

- Choose Approach A: capability-interface layering + a unified command catalog.
- Channels without platform registration support (for example, WhatsApp) only implement unified parse/execute behavior.
- Registration failure policy is fixed as non-blocking (warn + retry), without adding a strict-mode switch.

## Architecture Design

### 1) Unified Command Catalog

Add `pkg/commands` (command domain) as the only source of command definitions. The definition structure includes:

- `Name`: command name (for example, `help`).
- `Description`: description for platform menus.
- `Usage`: user-facing hint (for example, `/show [model|channel]`).
- `Aliases`: optional aliases.
- `Channels`: optional channel allow-list (empty means all channels).
- `Handler`: unified execution entry.

### 2) Unified Dispatcher

Add `CommandDispatcher`:

- Input: `CommandRequest` (channel/chat/sender/text/message_id, etc.).
- Output: `DispatchResult` (matched/executed/error).
- Semantics:
  - command matched: execute handler and return handled;
  - command not matched: return control to channel normal message flow (into agent).

### 3) Layered Channel Capability Interfaces

Add optional interfaces in `pkg/channels` (without changing the existing `Channel` main interface):

- `CommandParserCapable` (optional): indicates the channel can parse command entry points.
- `CommandRegistrarCapable` (optional): indicates the channel supports platform menu registration.

Telegram implements `CommandRegistrarCapable`; WhatsApp may omit this interface.

### 4) Telegram Adapter Layer

The Telegram channel performs two parallel responsibilities in `Start()`:

- start the message-processing pipeline (immediately available);
- run asynchronous command registration (does not block availability).

Registration data comes from the unified command catalog and is mapped to Telegram `BotCommand`.

## Startup Sequence and Data Flow

### Startup Sequence

1. Inject command definitions and dispatcher when creating channels.
2. `Start()` establishes the connection and starts message listening.
3. If the channel supports registration capability, asynchronously run `RegisterCommands()`.
4. On registration failure: log a warning and retry with backoff while keeping the channel running.

### Inbound Message Flow

1. The channel receives a text message.
2. Convert it into `CommandRequest` and call the dispatcher.
3. If matched: execute and reply.
4. If unmatched: continue the original flow into the agent.

### Platform Registration Flow

1. Filter visible commands from the unified command definitions by channel.
2. Convert to platform command structures and submit through platform APIs.
3. Mark registered on success; enter retry flow on failure.

## Error Handling and Observability

### Error Levels

- User input error: return usage text (not a system error).
- Command execution error: return a user-readable error plus error logs.
- Platform registration error: warning logs + automatic retries, without interrupting startup.

### Logging Recommendations

- `command registration started/succeeded/failed`
- `command dispatch matched/unmatched`
- `command execution succeeded/failed`

Recommended fields: `channel`, `command`, `attempt`, `next_retry_seconds`, `error`.

### Retry Strategy

- Exponential backoff: `5s -> 15s -> 60s -> 5m -> 10m(cap)`.
- Stop retrying immediately after success.
- `Stop()` must cancel retry goroutines to prevent leaks.

## Testing and Acceptance

### Test Scope

- Unit: registry uniqueness, channel filtering, dispatcher matching, and argument parsing.
- Integration: Telegram registration failure does not block startup; retries stop after a successful retry.
- Regression: existing `/help /start /show /list` behavior does not regress; non-command messages still flow to the agent.

### Acceptance Criteria

- Adding a new command requires changing only one unified definition.
- Telegram platform menu commands are synchronized automatically.
- Channels such as WhatsApp still use unified command parse/execute behavior without platform registration support.
- Registration failures do not block startup, and retry logs are observable.

## Risks and Mitigations

- Risk: command definitions and execution become too tightly coupled, making tests harder.
  - Mitigation: decouple command metadata from executors; inject executor dependencies via interfaces.
- Risk: channel adapter differences cause behavior drift.
  - Mitigation: cover channel-specific inputs with a unified dispatcher test matrix.
- Risk: retry logic leaks goroutines.
  - Mitigation: use a unified context lifecycle and `Stop()` cleanup tests.
