package telegram

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/commands"
)

func TestStartCommandRegistration_DoesNotBlock(t *testing.T) {
	ch := &TelegramChannel{}
	started := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch.registerFunc = func(context.Context, []commands.Definition) error {
		started <- struct{}{}
		return errors.New("temporary failure")
	}

	ch.startCommandRegistration(ctx, []commands.Definition{{Name: "help"}})

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("registration did not start asynchronously")
	}
}
