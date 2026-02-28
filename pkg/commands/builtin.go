package commands

import "github.com/sipeed/picoclaw/pkg/config"

func BuiltinDefinitions(_ *config.Config) []Definition {
	return []Definition{
		{
			Name:        "start",
			Description: "Start the bot",
			Usage:       "/start",
			Channels:    []string{"telegram", "whatsapp", "whatsapp_native"},
		},
		{
			Name:        "help",
			Description: "Show this help message",
			Usage:       "/help",
			Channels:    []string{"telegram", "whatsapp", "whatsapp_native"},
		},
		{
			Name:        "show",
			Description: "Show current configuration",
			Usage:       "/show [model|channel]",
			Channels:    []string{"telegram", "whatsapp", "whatsapp_native"},
		},
		{
			Name:        "list",
			Description: "List available options",
			Usage:       "/list [models|channels]",
			Channels:    []string{"telegram", "whatsapp", "whatsapp_native"},
		},
	}
}
