package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/sipeed/picoclaw/pkg/session"
)

func sessionCommand() Definition {
	return Definition{
		Name:        "session",
		Description: "Manage chat sessions",
		SubCommands: []SubCommand{
			{
				Name:        "list",
				Description: "List sessions for current chat",
				Handler:     sessionListHandler(),
			},
			{
				Name:        "resume",
				Description: "Resume a previous session",
				ArgsUsage:   "<index>",
				Handler:     sessionResumeHandler(),
			},
		},
	}
}

func sessionListHandler() Handler {
	return func(_ context.Context, req Request, rt *Runtime) error {
		if rt == nil || rt.SessionOps == nil || strings.TrimSpace(req.ScopeKey) == "" {
			return req.Reply(unavailableMsg)
		}

		list, err := rt.SessionOps.List(req.ScopeKey)
		if err != nil {
			return req.Reply(fmt.Sprintf("Failed to list sessions: %v", err))
		}
		if len(list) == 0 {
			return req.Reply("No sessions found for current chat.")
		}
		return req.Reply(formatSessionList(list))
	}
}

func sessionResumeHandler() Handler {
	return func(_ context.Context, req Request, rt *Runtime) error {
		if rt == nil || rt.SessionOps == nil || strings.TrimSpace(req.ScopeKey) == "" {
			return req.Reply(unavailableMsg)
		}

		// tokens: [/session, resume, <index>]
		indexStr := nthToken(req.Text, 2)
		if indexStr == "" {
			return req.Reply("Usage: /session resume <index>")
		}
		index, err := strconv.Atoi(indexStr)
		if err != nil || index < 1 {
			return req.Reply("Usage: /session resume <index>")
		}

		sessionKey, err := rt.SessionOps.Resume(req.ScopeKey, index)
		if err != nil {
			return req.Reply(fmt.Sprintf("Failed to resume session %d: %v", index, err))
		}
		return req.Reply(fmt.Sprintf("Resumed session %d: %s", index, sessionKey))
	}
}

func formatSessionList(list []session.SessionMeta) string {
	lines := make([]string, 0, len(list)+1)
	lines = append(lines, "Sessions for current chat:")
	for _, item := range list {
		activeMarker := " "
		if item.Active {
			activeMarker = "*"
		}
		updated := "-"
		if !item.UpdatedAt.IsZero() {
			updated = item.UpdatedAt.Format("2006-01-02 15:04")
		}
		lines = append(lines, fmt.Sprintf(
			"%d. [%s] %s | updated: %s | messages: %d",
			item.Ordinal, activeMarker, item.SessionKey, updated, item.MessageCnt,
		))
	}
	return strings.Join(lines, "\n")
}
