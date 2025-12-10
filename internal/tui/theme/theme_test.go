package theme

import "testing"

func withDetector(t *testing.T, detector func() bool) {
	original := detectDarkBackground
	detectDarkBackground = detector
	t.Cleanup(func() {
		detectDarkBackground = original
	})
}

func TestCurrentAutoUsesLightThemeWhenBackgroundIsLight(t *testing.T) {
	t.Setenv("NTM_THEME", "")
	withDetector(t, func() bool { return false })

	if got := Current(); got.Base != CatppuccinLatte.Base {
		t.Fatalf("expected light theme (Latte) for light background, got base %s", got.Base)
	}
}

func TestCurrentAutoUsesDarkThemeWhenBackgroundIsDark(t *testing.T) {
	t.Setenv("NTM_THEME", "")
	withDetector(t, func() bool { return true })

	if got := Current(); got.Base != CatppuccinMocha.Base {
		t.Fatalf("expected dark theme (Mocha) for dark background, got base %s", got.Base)
	}
}

func TestCurrentRespectsExplicitThemeOverrides(t *testing.T) {
	t.Setenv("NTM_THEME", "latte")
	withDetector(t, func() bool { return true })

	if got := Current(); got.Base != CatppuccinLatte.Base {
		t.Fatalf("expected Latte when explicitly requested, got base %s", got.Base)
	}

	t.Setenv("NTM_THEME", "mocha")
	withDetector(t, func() bool { return false })

	if got := Current(); got.Base != CatppuccinMocha.Base {
		t.Fatalf("expected Mocha when explicitly requested, got base %s", got.Base)
	}
}

func TestCurrentTreatsAutoValueAsDetection(t *testing.T) {
	t.Setenv("NTM_THEME", "auto")
	withDetector(t, func() bool { return false })

	if got := Current(); got.Base != CatppuccinLatte.Base {
		t.Fatalf("expected Latte for auto detection on light background, got base %s", got.Base)
	}
}
