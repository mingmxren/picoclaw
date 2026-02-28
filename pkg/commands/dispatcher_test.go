package commands

import (
	"context"
	"testing"
)

func TestDispatcher_MatchSlashCommand(t *testing.T) {
	called := false
	defs := []Definition{
		{
			Name: "help",
			Handler: func(context.Context, Request) error {
				called = true
				return nil
			},
		},
	}
	d := NewDispatcher(NewRegistry(defs))

	res := d.Dispatch(context.Background(), Request{
		Channel: "telegram",
		Text:    "/help",
	})
	if !res.Matched || !called || res.Err != nil {
		t.Fatalf("dispatch result = %+v, called=%v", res, called)
	}
}
