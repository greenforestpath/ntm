package ensemble

import "testing"

func TestSharing_ExportImportRoundTrip(t *testing.T) {
	input := map[string]any{"name": "shared-ensemble"}
	logTestStartSharing(t, input)

	preset := EnsemblePreset{
		Name:        "shared-ensemble",
		Description: "Shared preset",
		Modes:       []ModeRef{{ID: "deductive"}},
	}

	export := ExportFromPreset(preset)
	roundTrip := export.ToPreset()
	logTestResultSharing(t, roundTrip)

	assertEqualSharing(t, "name preserved", roundTrip.Name, preset.Name)
	assertEqualSharing(t, "mode count", len(roundTrip.Modes), 1)
}

func TestSharing_RemoteRejected(t *testing.T) {
	input := map[string]any{"mode": "nonexistent"}
	logTestStartSharing(t, input)

	catalog, err := LoadModeCatalog()
	assertNoErrorSharing(t, "load catalog", err)
	registry := NewEnsembleRegistry(EmbeddedEnsembles, catalog)

	export := EnsembleExport{
		SchemaVersion: EnsembleExportSchemaVersion,
		Name:          "bad-ensemble",
		Description:   "Bad",
		Modes:         []ModeRef{{ID: "does-not-exist"}},
	}

	err = export.Validate(catalog, registry)
	logTestResultSharing(t, err)
	assertTrueSharing(t, "validation error", err != nil)
}

func TestSharing_ChecksumValidation(t *testing.T) {
	input := map[string]any{"schema": 0}
	logTestStartSharing(t, input)

	catalog, err := LoadModeCatalog()
	assertNoErrorSharing(t, "load catalog", err)

	export := EnsembleExport{
		SchemaVersion: 0,
		Name:          "bad-schema",
		Description:   "Bad schema",
		Modes:         []ModeRef{{ID: "deductive"}},
	}

	err = export.Validate(catalog, NewEnsembleRegistry(EmbeddedEnsembles, catalog))
	logTestResultSharing(t, err)
	assertTrueSharing(t, "schema error", err != nil)
}

func logTestStartSharing(t *testing.T, input any) {
	t.Helper()
	t.Logf("TEST: %s - starting with input: %v", t.Name(), input)
}

func logTestResultSharing(t *testing.T, result any) {
	t.Helper()
	t.Logf("TEST: %s - got result: %v", t.Name(), result)
}

func assertNoErrorSharing(t *testing.T, desc string, err error) {
	t.Helper()
	t.Logf("TEST: %s - assertion: %s", t.Name(), desc)
	if err != nil {
		t.Fatalf("%s: %v", desc, err)
	}
}

func assertTrueSharing(t *testing.T, desc string, ok bool) {
	t.Helper()
	t.Logf("TEST: %s - assertion: %s", t.Name(), desc)
	if !ok {
		t.Fatalf("assertion failed: %s", desc)
	}
}

func assertEqualSharing(t *testing.T, desc string, got, want any) {
	t.Helper()
	t.Logf("TEST: %s - assertion: %s", t.Name(), desc)
	if got != want {
		t.Fatalf("%s: got %v want %v", desc, got, want)
	}
}
