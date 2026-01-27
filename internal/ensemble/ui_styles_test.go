package ensemble

import (
	"strings"
	"testing"
)

func TestModeBadge_AndTierChip(t *testing.T) {
	mode := ReasoningMode{ID: "deductive", Code: "A1", Category: CategoryFormal}
	badge := ModeBadge(mode)
	if badge == "" {
		t.Fatal("expected ModeBadge output")
	}
	if !strings.Contains(badge, "A1") {
		t.Fatalf("expected badge to include code, got %q", badge)
	}

	if TierChip("") != "" {
		t.Fatal("expected empty TierChip for empty tier")
	}
	if TierChip(TierCore) == "" {
		t.Fatal("expected TierChip output for core")
	}
}

func TestCategoryColor_DefaultAndASCII(t *testing.T) {
	_ = CategoryColor(CategoryFormal)
	_ = CategoryColor(CategoryAmpliative)
	_ = CategoryColor(CategoryUncertainty)
	_ = CategoryColor(CategoryVagueness)
	_ = CategoryColor(CategoryChange)
	_ = CategoryColor(CategoryCausal)
	_ = CategoryColor(CategoryPractical)
	_ = CategoryColor(CategoryStrategic)
	_ = CategoryColor(CategoryDialectical)
	_ = CategoryColor(CategoryModal)
	_ = CategoryColor(CategoryDomain)
	_ = CategoryColor(CategoryMeta)
	_ = CategoryColor("unknown")
	if !isASCII("abc") {
		t.Fatal("expected ascii to be true")
	}
	if isASCII("◆") {
		t.Fatal("expected unicode to be false")
	}
}
