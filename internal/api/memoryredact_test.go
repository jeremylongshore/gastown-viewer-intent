package api

import (
	"strings"
	"testing"

	"github.com/intent-solutions-io/gastown-viewer-intent/internal/model"
)

func TestRedactMemory_ClassA_PartnerNameSubstring(t *testing.T) {
	for _, name := range []string{"kobiton", "nixtla", "mudit", "polygon"} {
		t.Run(name, func(t *testing.T) {
			m := &model.Memory{Key: "test", Content: "engagement with " + name + " is ongoing"}
			RedactMemory(m, DefaultMemoryClassification, false)
			if !m.Redacted {
				t.Errorf("%q should have triggered redaction", name)
			}
			if strings.Contains(strings.ToLower(m.Content), name) {
				t.Errorf("redacted content still contains %q: %q", name, m.Content)
			}
			if !contains(m.RedactionMarkers, "partner-name") {
				t.Errorf("RedactionMarkers should include partner-name; got %v", m.RedactionMarkers)
			}
		})
	}
}

func TestRedactMemory_ClassA_CaseInsensitive(t *testing.T) {
	m := &model.Memory{Key: "k", Content: "Kobiton + KOBITON + KoBiToN are all partners"}
	RedactMemory(m, DefaultMemoryClassification, false)
	if !m.Redacted {
		t.Fatalf("case variants should all redact")
	}
	// Three occurrences should all be replaced.
	if strings.Contains(strings.ToLower(m.Content), "kobiton") {
		t.Errorf("at least one case variant survived: %q", m.Content)
	}
	count := strings.Count(m.Content, "[REDACTED partner-name]")
	if count != 3 {
		t.Errorf("expected 3 redaction markers, got %d in %q", count, m.Content)
	}
}

func TestRedactMemory_ClassA_WholeWordLit(t *testing.T) {
	// "lit" must redact "Lit Protocol" but NOT "literally" / "splitting" /
	// "literature". Without whole-word matching the redaction explodes
	// across normal English prose.
	cases := []struct {
		content      string
		shouldRedact bool
	}{
		{"working with Lit Protocol on KMS", true},
		{"lit on its own should also redact", true},
		{"we use this literally every day", false},
		{"splitting the workload", false},
		{"a study of literature was unrelated", false},
		{"lit. as an abbreviation", true}, // followed by `.` which is a boundary
	}
	for _, tc := range cases {
		t.Run(tc.content, func(t *testing.T) {
			m := &model.Memory{Key: "k", Content: tc.content}
			RedactMemory(m, DefaultMemoryClassification, false)
			if m.Redacted != tc.shouldRedact {
				t.Errorf("content=%q: expected Redacted=%v, got %v (output=%q)",
					tc.content, tc.shouldRedact, m.Redacted, m.Content)
			}
		})
	}
}

func TestRedactMemory_ClassA_WholeWordElm(t *testing.T) {
	cases := []struct {
		content      string
		shouldRedact bool
	}{
		{"engagement with elm consulting starts Q3", true},
		{"Elm at the helm", true},
		{"helm chart for the deploy", false},
		{"realm boundary issue", false},
		{"elmer fudd was the inspiration", false}, // "elmer" not "elm"
	}
	for _, tc := range cases {
		t.Run(tc.content, func(t *testing.T) {
			m := &model.Memory{Key: "k", Content: tc.content}
			RedactMemory(m, DefaultMemoryClassification, false)
			if m.Redacted != tc.shouldRedact {
				t.Errorf("content=%q: expected Redacted=%v, got %v (output=%q)",
					tc.content, tc.shouldRedact, m.Redacted, m.Content)
			}
		})
	}
}

func TestRedactMemory_ClassB_SecretPrefixes(t *testing.T) {
	// Each prefix gets a representative key of length >= 16. The redaction
	// must catch every one.
	tokens := map[string]string{
		"sk-":     "sk-1234567890abcdefghij",
		"sk-ant-": "sk-ant-api03-abcdefghijklmnop",
		"ghp_":    "ghp_aaaaaaaaaaaaaaaaaaaa",
		"gho_":    "gho_aaaaaaaaaaaaaaaaaaaa",
		"ghs_":    "ghs_aaaaaaaaaaaaaaaaaaaa",
		"ghr_":    "ghr_aaaaaaaaaaaaaaaaaaaa",
		"AKIA":    "AKIAIOSFODNN7EXAMPLE",
		"ASIA":    "ASIAIOSFODNN7EXAMPLE",
		"glpat-":  "glpat-12345678901234567890",
		"tlk_":    "tlk_1234567890abcdef1234",
	}
	for prefix, tok := range tokens {
		t.Run(prefix, func(t *testing.T) {
			m := &model.Memory{Key: "k", Content: "the key is " + tok + " do not share"}
			RedactMemory(m, DefaultMemoryClassification, false)
			if !m.Redacted {
				t.Errorf("token %q (prefix %q) should redact", tok, prefix)
			}
			if strings.Contains(m.Content, tok) {
				t.Errorf("token %q still appears in output %q", tok, m.Content)
			}
			if !contains(m.RedactionMarkers, "secret") {
				t.Errorf("RedactionMarkers should include secret; got %v", m.RedactionMarkers)
			}
		})
	}
}

func TestRedactMemory_ClassB_ShortPrefixesNotRedacted(t *testing.T) {
	// Acronym-style prefix use must NOT trigger redaction. Length floor
	// (secretRunMinLength = 16) is the defense.
	cases := []string{
		"we use sk- as a shorthand",
		"the AKIA cloud team",
		"split on ghp_, gho_, and ghs_ tokens",
		"glpat- is the prefix",
	}
	for _, content := range cases {
		t.Run(content, func(t *testing.T) {
			m := &model.Memory{Key: "k", Content: content}
			RedactMemory(m, DefaultMemoryClassification, false)
			if m.Redacted {
				t.Errorf("short prefix-only mention should not redact: %q → %q", content, m.Content)
			}
		})
	}
}

func TestRedactMemory_LongerPrefixWinsAgainstShorter(t *testing.T) {
	// sk-ant- is in the prefix list BEFORE sk-. The longer match should
	// consume the run so the prefix attribution is correct even though
	// the marker class is the same ("secret").
	m := &model.Memory{Key: "k", Content: "key: sk-ant-api03-ZZZZZZZZZZZZZZZZ"}
	RedactMemory(m, DefaultMemoryClassification, false)
	if !m.Redacted {
		t.Fatalf("sk-ant- key should redact")
	}
	if strings.Contains(m.Content, "sk-ant") {
		t.Errorf("sk-ant- not fully redacted: %q", m.Content)
	}
}

func TestRedactMemory_RevealPassthrough(t *testing.T) {
	m := &model.Memory{Key: "k", Content: "kobiton + sk-1234567890abcdef + lit protocol"}
	original := m.Content
	RedactMemory(m, DefaultMemoryClassification, true)
	if m.Redacted {
		t.Errorf("reveal=true should not mark Redacted")
	}
	if m.Content != original {
		t.Errorf("reveal=true must passthrough; got %q (original %q)", m.Content, original)
	}
	if len(m.RedactionMarkers) != 0 {
		t.Errorf("reveal=true should not populate markers; got %v", m.RedactionMarkers)
	}
}

func TestRedactMemory_NoMatchesUnchanged(t *testing.T) {
	m := &model.Memory{Key: "k", Content: "completely unrelated notes about CI flakiness"}
	RedactMemory(m, DefaultMemoryClassification, false)
	if m.Redacted {
		t.Errorf("clean content should not be marked Redacted")
	}
	if m.Content != "completely unrelated notes about CI flakiness" {
		t.Errorf("clean content should be unmodified; got %q", m.Content)
	}
}

func TestRedactMemory_MultipleClassesProduceMultipleMarkers(t *testing.T) {
	m := &model.Memory{Key: "k", Content: "kobiton engagement + AKIAIOSFODNN7EXAMPLE for AWS"}
	RedactMemory(m, DefaultMemoryClassification, false)
	if !m.Redacted {
		t.Fatalf("both classes present should redact")
	}
	if !contains(m.RedactionMarkers, "partner-name") {
		t.Errorf("missing partner-name marker; got %v", m.RedactionMarkers)
	}
	if !contains(m.RedactionMarkers, "secret") {
		t.Errorf("missing secret marker; got %v", m.RedactionMarkers)
	}
	// Markers must be deterministically ordered (alphabetical).
	if len(m.RedactionMarkers) >= 2 && m.RedactionMarkers[0] != "partner-name" {
		t.Errorf("markers not alphabetical; got %v", m.RedactionMarkers)
	}
}

func TestRedactMemory_NilMemory(t *testing.T) {
	// Defensive: must not panic on a nil pointer. Caller will pass *Memory
	// directly from slice indexing where this can't happen in practice,
	// but a future map iteration could pass nil.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("nil memory caused panic: %v", r)
		}
	}()
	RedactMemory(nil, DefaultMemoryClassification, false)
}

func TestRedactMemories_AppliesPerEntry(t *testing.T) {
	memories := []model.Memory{
		{Key: "a", Content: "kobiton partnership"},
		{Key: "b", Content: "clean note"},
		{Key: "c", Content: "key: ghp_aaaaaaaaaaaaaaaa"},
	}
	out := RedactMemories(memories, DefaultMemoryClassification, false)
	if &out[0] != &memories[0] {
		t.Error("RedactMemories should mutate in place and return the same backing slice")
	}
	if !out[0].Redacted {
		t.Error("entry 0 should redact (partner)")
	}
	if out[1].Redacted {
		t.Error("entry 1 should be clean")
	}
	if !out[2].Redacted {
		t.Error("entry 2 should redact (secret)")
	}
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
