package ensemble

import "testing"

func TestModeCards_AllModesPresent(t *testing.T) {
	logTestStartModeCards(t, "embedded catalog")

	catalog, err := LoadModeCatalog()
	logTestResultModeCards(t, err)
	assertNoErrorModeCards(t, "load catalog", err)

	missing := 0
	for _, mode := range EmbeddedModes {
		card, err := catalog.GetModeCard(mode.ID)
		if err != nil || card == nil {
			missing++
		}
	}

	assertEqualModeCards(t, "missing cards", missing, 0)
}

func TestModeCards_RequiredFields(t *testing.T) {
	logTestStartModeCards(t, "required fields")

	catalog, err := LoadModeCatalog()
	assertNoErrorModeCards(t, "load catalog", err)

	card, err := catalog.GetModeCard("deductive")
	logTestResultModeCards(t, card)
	assertNoErrorModeCards(t, "get mode card", err)

	assertTrueModeCards(t, "mode id present", card.ModeID != "")
	assertTrueModeCards(t, "name present", card.Name != "")
	assertTrueModeCards(t, "category present", card.Category != "")
	assertTrueModeCards(t, "tier present", card.Tier != "")
}

func TestModeCards_CategoryConsistency(t *testing.T) {
	logTestStartModeCards(t, "category consistency")

	catalog, err := LoadModeCatalog()
	assertNoErrorModeCards(t, "load catalog", err)

	mode := catalog.GetMode("deductive")
	card, err := catalog.GetModeCard("deductive")
	logTestResultModeCards(t, map[string]any{"mode": mode, "card": card})
	assertNoErrorModeCards(t, "get card", err)

	assertEqualModeCards(t, "category matches", card.Category, mode.Category)
}

func logTestStartModeCards(t *testing.T, input any) {
	t.Helper()
	t.Logf("TEST: %s - starting with input: %v", t.Name(), input)
}

func logTestResultModeCards(t *testing.T, result any) {
	t.Helper()
	t.Logf("TEST: %s - got result: %v", t.Name(), result)
}

func assertNoErrorModeCards(t *testing.T, desc string, err error) {
	t.Helper()
	t.Logf("TEST: %s - assertion: %s", t.Name(), desc)
	if err != nil {
		t.Fatalf("%s: %v", desc, err)
	}
}

func assertTrueModeCards(t *testing.T, desc string, ok bool) {
	t.Helper()
	t.Logf("TEST: %s - assertion: %s", t.Name(), desc)
	if !ok {
		t.Fatalf("assertion failed: %s", desc)
	}
}

func assertEqualModeCards(t *testing.T, desc string, got, want any) {
	t.Helper()
	t.Logf("TEST: %s - assertion: %s", t.Name(), desc)
	if got != want {
		t.Fatalf("%s: got %v want %v", desc, got, want)
	}
}
