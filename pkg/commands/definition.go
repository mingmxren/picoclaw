package commands

// Definition is the single-source metadata and behavior contract for a slash command.
//
// Design notes (phase 1):
// - Every channel reads command shape from this type instead of keeping local copies.
// - Visibility is global: all definitions are considered available to all channels.
// - Platform menu registration (for example Telegram BotCommand) also derives from this
//   same definition so UI labels and runtime behavior stay aligned.
type Definition struct {
	Name        string
	Description string
	Usage       string
	Aliases     []string
	Handler     Handler
}
