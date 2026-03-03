package commands

import "testing"

func TestRegistry_Definitions_ReturnsCopy(t *testing.T) {
	defs := []Definition{
		{Name: "help", Description: "Show help"},
		{Name: "admin", Description: "Admin command"},
	}
	r := NewRegistry(defs)

	got := r.Definitions()
	if len(got) != 2 {
		t.Fatalf("definitions len = %d, want 2", len(got))
	}

	got[0].Name = "mutated"
	again := r.Definitions()
	if again[0].Name != "help" {
		t.Fatalf("registry should not be mutated by caller, got first name %q", again[0].Name)
	}
}
