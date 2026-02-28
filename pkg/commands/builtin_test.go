package commands

import "testing"

func TestBuiltinDefinitions_ContainsTelegramDefaults(t *testing.T) {
	defs := BuiltinDefinitions(nil)
	names := map[string]bool{}
	for _, d := range defs {
		names[d.Name] = true
	}
	for _, want := range []string{"help", "start", "show", "list"} {
		if !names[want] {
			t.Fatalf("missing command %q", want)
		}
	}
}
