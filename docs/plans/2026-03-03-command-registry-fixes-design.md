# Command Registry Fixes Design

Date: 2026-03-03
Branch: `codex/pr-split-1-command-registry`
Phase: 1/2 command centralization — post-review fixes

## Context

Code review of commit `ffc063e` identified 5 Important issues and 4 Minor issues.
This document covers the design for fixing all 5 Important issues.

## Issues Summary

| # | Issue | Root Cause |
|---|-------|-----------|
| 1 | Executor/Registry rebuilt per message | `handleCommand` calls `NewExecutor(NewRegistry(...))` on every invocation |
| 2 | `/show model` reads config instead of runtime agent | Handler in commands pkg cannot access agent registry |
| 3 | `/show agents` and `/list agents` silently removed | New builtin definitions omit agents sub-commands |
| 4 | `contains()` dead code in dispatcher.go | Unused helper left from development |
| 5 | Telegram double command registration | Both `initBotCommands` and `startCommandRegistration` call SetMyCommands |

## Design Decisions

### Architecture: SubCommand pattern + Deps struct + command-group files

**Key choices:**

1. **SubCommand struct** — Commands like `/show` and `/list` declare sub-commands structurally. Usage strings are auto-generated from sub-command names, making drift impossible.

2. **Deps struct** — A `commands.Deps` struct with function fields provides runtime data (model info, agent IDs, enabled channels) without importing agent/channels packages.

3. **Command-group files** — Each command group lives in its own file (`cmd_show.go`, `cmd_list.go`, etc.) within `pkg/commands/`. Metadata and handler are co-located. Agent package only wires Deps once.

4. **Executor caching** — The Executor is created once in `NewAgentLoop` and stored as a field, eliminating per-message allocation.

### Why not alternatives

- **Callback struct without SubCommand**: metadata and handler can drift (usage says `[model|channel]` but handler supports more).
- **Interface injection**: adding methods is a breaking change in Go; Deps struct field additions are not.
- **Handlers in agent package**: AgentLoop becomes a God Object as commands grow.

## Detailed Design

### 1. New types in commands package

#### `SubCommand` (definition.go)

```go
type SubCommand struct {
    Name        string
    Description string
    ArgsUsage   string  // optional, e.g. "<session-id>"
    Handler     Handler
}
```

#### Updated `Definition` (definition.go)

```go
type Definition struct {
    Name        string
    Description string
    Usage       string        // for simple commands; auto-generated when SubCommands present
    Aliases     []string
    SubCommands []SubCommand  // optional
    Handler     Handler       // for simple commands (no sub-commands)
}

func (d Definition) EffectiveUsage() string {
    if len(d.SubCommands) == 0 {
        return d.Usage
    }
    names := make([]string, 0, len(d.SubCommands))
    for _, sc := range d.SubCommands {
        name := sc.Name
        if sc.ArgsUsage != "" {
            name += " " + sc.ArgsUsage
        }
        names = append(names, name)
    }
    return fmt.Sprintf("/%s [%s]", d.Name, strings.Join(names, "|"))
}
```

#### `Deps` (deps.go)

```go
type Deps struct {
    Config             *config.Config
    GetModelInfo       func() (name, provider string)
    ListAgentIDs       func() []string
    GetEnabledChannels func() []string
}
```

### 2. Executor sub-command routing (executor.go)

When a Definition has SubCommands, the Executor:
1. Parses the sub-command name from the second token
2. Matches against SubCommand.Name
3. Passes the Request (with full text) to the matched SubCommand.Handler
4. If no sub-command token or no match, replies with auto-generated usage

```go
func (e *Executor) executeDefinition(ctx context.Context, req Request, def Definition) ExecuteResult {
    if len(def.SubCommands) == 0 {
        // Simple command — use def.Handler directly
        if def.Handler == nil {
            return ExecuteResult{Outcome: OutcomePassthrough, Command: def.Name}
        }
        err := def.Handler(ctx, req)
        return ExecuteResult{Outcome: OutcomeHandled, Command: def.Name, Err: err}
    }

    // Sub-command routing
    subName := secondToken(req.Text)
    if subName == "" {
        // No sub-command provided — reply with usage
        if req.Reply != nil {
            _ = req.Reply("Usage: " + def.EffectiveUsage())
        }
        return ExecuteResult{Outcome: OutcomeHandled, Command: def.Name}
    }

    for _, sc := range def.SubCommands {
        if normalizeCommandName(sc.Name) == normalizeCommandName(subName) {
            if sc.Handler == nil {
                return ExecuteResult{Outcome: OutcomePassthrough, Command: def.Name}
            }
            err := sc.Handler(ctx, req)
            return ExecuteResult{Outcome: OutcomeHandled, Command: def.Name, Err: err}
        }
    }

    // Unknown sub-command
    if req.Reply != nil {
        _ = req.Reply(fmt.Sprintf("Unknown parameter: %s. Usage: %s", subName, def.EffectiveUsage()))
    }
    return ExecuteResult{Outcome: OutcomeHandled, Command: def.Name}
}
```

### 3. Command-group files

#### cmd_show.go

```go
func showCommand(deps *Deps) Definition {
    return Definition{
        Name:        "show",
        Description: "Show current configuration",
        SubCommands: []SubCommand{
            {
                Name:        "model",
                Description: "Current model and provider",
                Handler: func(_ context.Context, req Request) error {
                    name, provider := deps.GetModelInfo()
                    return req.Reply(fmt.Sprintf("Current Model: %s (Provider: %s)", name, provider))
                },
            },
            {
                Name:        "channel",
                Description: "Current channel",
                Handler: func(_ context.Context, req Request) error {
                    return req.Reply(fmt.Sprintf("Current Channel: %s", req.Channel))
                },
            },
            {
                Name:        "agents",
                Description: "Registered agents",
                Handler: func(_ context.Context, req Request) error {
                    ids := deps.ListAgentIDs()
                    if len(ids) == 0 {
                        return req.Reply("No agents registered")
                    }
                    return req.Reply(fmt.Sprintf("Registered agents: %s", strings.Join(ids, ", ")))
                },
            },
        },
    }
}
```

#### cmd_list.go

```go
func listCommand(deps *Deps) Definition {
    return Definition{
        Name:        "list",
        Description: "List available options",
        SubCommands: []SubCommand{
            {
                Name:        "models",
                Description: "Configured models",
                Handler: func(_ context.Context, req Request) error { ... },
            },
            {
                Name:        "channels",
                Description: "Enabled channels",
                Handler: func(_ context.Context, req Request) error {
                    enabled := deps.GetEnabledChannels()
                    return req.Reply(fmt.Sprintf("Enabled Channels:\n- %s", strings.Join(enabled, "\n- ")))
                },
            },
            {
                Name:        "agents",
                Description: "Registered agents",
                Handler: func(_ context.Context, req Request) error {
                    ids := deps.ListAgentIDs()
                    return req.Reply(fmt.Sprintf("Registered agents: %s", strings.Join(ids, ", ")))
                },
            },
        },
    }
}
```

#### cmd_start.go, cmd_help.go

Simple commands without sub-commands, using `Handler` directly.

#### builtin.go (simplified to aggregation only)

```go
func BuiltinDefinitions(deps *Deps) []Definition {
    return []Definition{
        startCommand(),
        helpCommand(deps),
        showCommand(deps),
        listCommand(deps),
    }
}
```

`enabledChannels()` helper is removed; `/list channels` uses `deps.GetEnabledChannels()`.

### 4. Agent loop wiring (loop.go)

```go
func NewAgentLoop(cfg, registry, channelMgr, ...) *AgentLoop {
    al := &AgentLoop{ ... }

    deps := &commands.Deps{
        Config: cfg,
        GetModelInfo: func() (string, string) {
            agent := registry.GetDefaultAgent()
            if agent == nil {
                return cfg.Agents.Defaults.GetModelName(), cfg.Agents.Defaults.Provider
            }
            return agent.Model, cfg.Agents.Defaults.Provider
        },
        ListAgentIDs:       registry.ListAgentIDs,
        GetEnabledChannels: channelMgr.GetEnabledChannels,
    }
    al.cmdExecutor = commands.NewExecutor(
        commands.NewRegistry(commands.BuiltinDefinitions(deps)),
    )

    return al
}
```

`handleCommand` becomes:

```go
func (al *AgentLoop) handleCommand(ctx context.Context, msg bus.InboundMessage) (string, bool) {
    content := strings.TrimSpace(msg.Content)
    if !strings.HasPrefix(content, "/") {
        return "", false
    }

    var commandReply string
    result := al.cmdExecutor.Execute(ctx, commands.Request{
        Channel:  msg.Channel,
        ChatID:   msg.ChatID,
        SenderID: msg.SenderID,
        Text:     msg.Content,
        Reply: func(text string) error {
            commandReply = text
            return nil
        },
    })

    // ... same switch as before
}
```

### 5. Telegram: remove initBotCommands

- Delete `initBotCommands` method entirely.
- `startCommandRegistration` is the sole registration path.
- Definitions are passed via `commands.NewRegistry(commands.BuiltinDefinitions(deps)).Definitions()`.
  - Note: Telegram needs a Deps with at minimum a Config. Since Telegram doesn't execute handlers (only reads Name/Description for registration), nil function fields are acceptable.

### 6. Dead code removal

- Delete `contains()` from `dispatcher.go`.
- Delete `enabledChannels()` from `builtin.go` (replaced by `deps.GetEnabledChannels()`).
- Delete `commandArgs()` from `builtin.go` (replaced by Executor sub-command routing).

## File Change Summary

| File | Action |
|------|--------|
| `pkg/commands/definition.go` | Add SubCommand, ArgsUsage, EffectiveUsage |
| `pkg/commands/deps.go` | **New** — Deps struct |
| `pkg/commands/executor.go` | Add sub-command routing logic |
| `pkg/commands/dispatcher.go` | Remove `contains()`, add `secondToken()` |
| `pkg/commands/cmd_start.go` | **New** — extracted from builtin.go |
| `pkg/commands/cmd_help.go` | **New** — extracted from builtin.go |
| `pkg/commands/cmd_show.go` | **New** — with agents sub-command restored |
| `pkg/commands/cmd_list.go` | **New** — with agents sub-command restored |
| `pkg/commands/builtin.go` | Simplified to aggregation only |
| `pkg/agent/loop.go` | Wire Deps in NewAgentLoop, cache cmdExecutor, simplify handleCommand |
| `pkg/channels/telegram/telegram.go` | Remove `initBotCommands` |
| Tests | Update existing + add sub-command routing tests |

## Extensibility

Adding `/session list` and `/session resume <id>` in Phase 2:

1. Add fields to Deps: `ListSessions`, `ResumeSession`
2. Create `cmd_session.go` with sub-commands
3. Add `sessionCommand(deps)` to `BuiltinDefinitions`
4. Wire new Deps fields in agent loop

No existing file changes required except `builtin.go` (one line) and `loop.go` (Deps wiring).
