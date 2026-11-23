package riqu

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestBindValuesConversions(t *testing.T) {
	var target struct {
		Text     string
		Flag     bool
		Count    *int
		Unsigned uint16
		Ratio    float64
		Numbers  []int
		Skip     string `cookie:"-"`
	}
	err := bindValues(&target, "cookie", TValues{
		"Text": {"hello"}, "Flag": {"true"}, "Count": {"12"},
		"Unsigned": {"42"}, "Ratio": {"1.5"}, "Numbers": {"2", "3"}, "Skip": {"ignored"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if target.Text != "hello" || !target.Flag || target.Count == nil || *target.Count != 12 || target.Unsigned != 42 || target.Ratio != 1.5 || len(target.Numbers) != 2 || target.Numbers[1] != 3 || target.Skip != "" {
		t.Fatalf("unexpected binding result: %+v", target)
	}
}

func TestBindValuesInvalidTargetsAndConversion(t *testing.T) {
	var nilTarget *struct{}
	for _, target := range []any{nil, nilTarget, struct{}{}, new(string)} {
		if err := bindValues(target, "header", nil); err == nil || !strings.HasPrefix(err.Error(), "riqu: ") {
			t.Fatalf("expected a prefixed binding error, got %v", err)
		}
	}
	var target struct {
		Count int8 `header:"x-count"`
	}
	err := bindValues(&target, "header", TValues{"X-Count": {"128"}})
	var parseErr *strconv.NumError
	if !errors.As(err, &parseErr) || !strings.Contains(err.Error(), "Count") {
		t.Fatalf("expected a wrapped conversion error with field context, got %v", err)
	}
	var unsupported struct{ Value map[string]string }
	if err := bindValues(&unsupported, "cookie", TValues{"Value": {"x"}}); err == nil {
		t.Fatal("expected unsupported field type to fail")
	}
}
