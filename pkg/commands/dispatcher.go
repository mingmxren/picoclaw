package commands

import (
	"context"
	"strings"
)

type Handler func(ctx context.Context, req Request) error

type Request struct {
	Channel   string
	ChatID    string
	SenderID  string
	Text      string
	MessageID string
	Reply     func(text string) error
}

type Result struct {
	Matched bool
	Handled bool
	Command string
	Err     error
}

type Dispatcher struct {
	reg *Registry
}

type Dispatching interface {
	Dispatch(ctx context.Context, req Request) Result
}

type DispatchFunc func(ctx context.Context, req Request) Result

func (f DispatchFunc) Dispatch(ctx context.Context, req Request) Result {
	return f(ctx, req)
}

var commandPrefixes = []string{"/", "!"}

// NewDispatcher binds the unified parser/executor flow to one command registry.
func NewDispatcher(reg *Registry) *Dispatcher {
	return &Dispatcher{reg: reg}
}

// Dispatch parses slash commands and executes handlers from the shared registry.
// Unmatched messages intentionally return Matched=false so callers can fall back
// to normal agent message handling.
func (d *Dispatcher) Dispatch(ctx context.Context, req Request) Result {
	cmdName, ok := parseCommandName(req.Text)
	if !ok {
		return Result{Matched: false}
	}

	for _, def := range d.reg.Definitions() {
		if !matchesCommand(def, cmdName) {
			continue
		}
		if def.Handler == nil {
			// Definition-only command (for menu registration / discovery).
			// Let the inbound message continue to the agent loop.
			return Result{Matched: false, Handled: false, Command: def.Name}
		}
		err := def.Handler(ctx, req)
		return Result{Matched: true, Handled: true, Command: def.Name, Err: err}
	}

	return Result{Matched: false}
}

func firstToken(input string) string {
	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// parseCommandName accepts "/name", "!name", and Telegram's "/name@bot", then
// normalizes to lowercase command names.
func parseCommandName(input string) (string, bool) {
	token := firstToken(input)
	if token == "" {
		return "", false
	}

	name, ok := trimCommandPrefix(token)
	if !ok {
		return "", false
	}
	if i := strings.Index(name, "@"); i >= 0 {
		name = name[:i]
	}
	name = normalizeCommandName(name)
	if name == "" {
		return "", false
	}
	return name, true
}

func trimCommandPrefix(token string) (string, bool) {
	for _, prefix := range commandPrefixes {
		if strings.HasPrefix(token, prefix) {
			return strings.TrimPrefix(token, prefix), true
		}
	}
	return "", false
}

func normalizeCommandName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func matchesCommand(def Definition, cmdName string) bool {
	if normalizeCommandName(def.Name) == cmdName {
		return true
	}
	for _, alias := range def.Aliases {
		if normalizeCommandName(alias) == cmdName {
			return true
		}
	}
	return false
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
