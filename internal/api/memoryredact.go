package api

import (
	"regexp"
	"sort"
	"strings"

	"github.com/intent-solutions-io/gastown-viewer-intent/internal/model"
)

// MemoryClassificationPolicy is the runtime view of the rules documented
// in 000-docs/005-PP-POLICY-memories-classification-2026-05-24.md. Tests
// instantiate it directly; production uses DefaultMemoryClassification.
type MemoryClassificationPolicy struct {
	// PartnerNames are Class A substrings. Each entry is normalized to
	// lower-case at construction time; the matcher lower-cases the
	// content before testing.
	PartnerNames []string

	// PartnerNamesWhole are Class A entries that must match as
	// whole-word tokens — `lit` should redact "lit protocol" but not
	// "literally", "elm" should redact "elm consulting" but not "helm".
	PartnerNamesWhole []string

	// SecretPrefixes are Class B token-prefix patterns. The matcher
	// flags any non-whitespace run of length >= 16 that begins with one
	// of these prefixes. Prefixes are case-sensitive (their canonical
	// issuers' formats are case-sensitive).
	SecretPrefixes []string
}

// DefaultMemoryClassification is the production policy. Keep this in lock-
// step with the published policy doc; the audit-harness pre-commit hook
// hash-pins the doc, so policy edits + this constant move in the same PR
// by design.
var DefaultMemoryClassification = MemoryClassificationPolicy{
	PartnerNames: []string{
		"kobiton",
		"nixtla",
		"mudit",
		"polygon",
	},
	PartnerNamesWhole: []string{
		"lit",
		"elm",
	},
	SecretPrefixes: []string{
		"sk-ant-", // Anthropic — must come before "sk-" so it claims the longer prefix
		"sk-",     // OpenAI / Anthropic legacy
		"ghp_",    // GitHub PAT
		"gho_",    // GitHub OAuth
		"ghs_",    // GitHub App server-to-server
		"ghr_",    // GitHub refresh
		"AKIA",    // AWS long-term key ID
		"ASIA",    // AWS STS temporary key ID
		"glpat-",  // GitLab PAT
		"tlk_",    // Twilio
	},
}

// secretRunMinLength is the floor on a token-suspected match to avoid
// flagging prose where one of the prefixes appears acronym-style. Tokens
// in the wild are typically 32-64 chars; 16 lets us catch shortened test
// keys while staying above the false-positive threshold of "sk-it" or
// "we use ghp_for tokens".
const secretRunMinLength = 16

// whole-word boundary regex compiled once per policy lookup. Word
// characters are letters, digits, and underscore; the policy treats anything
// else as a boundary.
var wholeWordSepRE = regexp.MustCompile(`[^A-Za-z0-9_]`)

// RedactMemory applies the policy to a single Memory in-place: rewrites
// Content with `[REDACTED ...]` placeholders for every match and
// populates Redacted + RedactionMarkers. The result is safe to serialize
// to JSON and ship to the browser; the original bytes are discarded.
//
// reveal=true is a passthrough — no redaction, no marker population. The
// reveal path is the engineer's explicit override; the daemon logs the
// reveal event (path TBD per THREAT_MODEL.md follow-ups), but this function
// does not — its single responsibility is the redaction transform.
func RedactMemory(m *model.Memory, policy MemoryClassificationPolicy, reveal bool) {
	if m == nil || reveal {
		return
	}
	originalContent := m.Content
	content := originalContent
	markers := make(map[string]struct{})

	// Class A — partner-name substrings.
	contentLower := strings.ToLower(content)
	for _, name := range policy.PartnerNames {
		nameLower := strings.ToLower(name)
		if strings.Contains(contentLower, nameLower) {
			content = redactSubstringCI(content, name)
			markers["partner-name"] = struct{}{}
			contentLower = strings.ToLower(content)
		}
	}

	// Class A whole-word — match only as standalone tokens.
	for _, name := range policy.PartnerNamesWhole {
		if containsWholeWordCI(content, name) {
			content = redactWholeWordCI(content, name)
			markers["partner-name"] = struct{}{}
		}
	}

	// Class B — secret-prefix runs. Walk the content once, when we hit a
	// prefix match, consume up to the next whitespace boundary and check
	// the run length floor.
	content = redactSecretRuns(content, policy.SecretPrefixes, markers)

	if content != originalContent {
		m.Content = content
		m.Redacted = true
		m.RedactionMarkers = sortedMarkers(markers)
	}
}

// RedactMemories applies RedactMemory to every entry in a slice. Returns
// the input slice (modified in place) for chaining convenience.
func RedactMemories(memories []model.Memory, policy MemoryClassificationPolicy, reveal bool) []model.Memory {
	for i := range memories {
		RedactMemory(&memories[i], policy, reveal)
	}
	return memories
}

// containsWholeWordCI is a case-insensitive whole-word containment check.
// "lit" matches in "lit protocol" but not in "literally" or "splitting".
func containsWholeWordCI(s, word string) bool {
	if word == "" {
		return false
	}
	lower := strings.ToLower(s)
	wordLower := strings.ToLower(word)
	start := 0
	for {
		idx := strings.Index(lower[start:], wordLower)
		if idx < 0 {
			return false
		}
		abs := start + idx
		// Check left boundary
		if abs > 0 && !isWordBoundary(rune(lower[abs-1])) {
			start = abs + 1
			continue
		}
		// Check right boundary
		endAbs := abs + len(wordLower)
		if endAbs < len(lower) && !isWordBoundary(rune(lower[endAbs])) {
			start = abs + 1
			continue
		}
		return true
	}
}

// redactWholeWordCI replaces every whole-word case-insensitive occurrence
// of word in s with `[REDACTED partner-name]`. Word casing in the source is
// preserved up to the redaction marker.
func redactWholeWordCI(s, word string) string {
	if word == "" {
		return s
	}
	var out strings.Builder
	out.Grow(len(s))
	lower := strings.ToLower(s)
	wordLower := strings.ToLower(word)
	wordLen := len(wordLower)
	i := 0
	for i < len(s) {
		idx := strings.Index(lower[i:], wordLower)
		if idx < 0 {
			out.WriteString(s[i:])
			break
		}
		abs := i + idx
		// Left boundary
		if abs > 0 && !isWordBoundary(rune(lower[abs-1])) {
			out.WriteString(s[i : abs+1])
			i = abs + 1
			continue
		}
		// Right boundary
		endAbs := abs + wordLen
		if endAbs < len(s) && !isWordBoundary(rune(lower[endAbs])) {
			out.WriteString(s[i : abs+1])
			i = abs + 1
			continue
		}
		out.WriteString(s[i:abs])
		out.WriteString("[REDACTED partner-name]")
		i = endAbs
	}
	return out.String()
}

// redactSubstringCI replaces every case-insensitive occurrence of needle
// in s with `[REDACTED partner-name]`. Does NOT respect word boundaries —
// this is the Class A substring path for unambiguous partner names that
// don't appear as common English words.
func redactSubstringCI(s, needle string) string {
	if needle == "" {
		return s
	}
	var out strings.Builder
	out.Grow(len(s))
	lower := strings.ToLower(s)
	needleLower := strings.ToLower(needle)
	needleLen := len(needleLower)
	i := 0
	for i < len(s) {
		idx := strings.Index(lower[i:], needleLower)
		if idx < 0 {
			out.WriteString(s[i:])
			break
		}
		abs := i + idx
		out.WriteString(s[i:abs])
		out.WriteString("[REDACTED partner-name]")
		i = abs + needleLen
	}
	return out.String()
}

// redactSecretRuns finds non-whitespace runs of length >= secretRunMinLength
// beginning with any of the configured prefixes and replaces them with
// `[REDACTED secret]`. Prefix order matters: place specific prefixes
// before generic ones (sk-ant- before sk-) so the longest-matching prefix
// claims the run.
func redactSecretRuns(s string, prefixes []string, markers map[string]struct{}) string {
	if len(prefixes) == 0 {
		return s
	}
	var out strings.Builder
	out.Grow(len(s))
	i := 0
	for i < len(s) {
		matched := false
		for _, p := range prefixes {
			if strings.HasPrefix(s[i:], p) {
				// Find end of the run (next whitespace or EOS).
				end := i + len(p)
				for end < len(s) && !isWhitespaceByte(s[end]) {
					end++
				}
				if end-i >= secretRunMinLength {
					out.WriteString("[REDACTED secret]")
					markers["secret"] = struct{}{}
					i = end
					matched = true
					break
				}
			}
		}
		if matched {
			continue
		}
		out.WriteByte(s[i])
		i++
	}
	return out.String()
}

// isWordBoundary reports whether r is a non-word character per the policy.
// Word characters are A-Za-z0-9_; everything else is a boundary.
func isWordBoundary(r rune) bool {
	return !((r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '_')
}

// isWhitespaceByte is the ASCII whitespace check used to terminate secret
// token runs. Aligned with bytes.TrimSpace's notion of whitespace for the
// runes the policy is realistically going to see.
func isWhitespaceByte(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f'
}

// sortedMarkers returns the marker set as a deterministic-ordered slice
// (stable across JSON serializations). Set ordering would otherwise be
// random and the JSON-encoded response would diff against itself.
func sortedMarkers(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// EvaluateMemoryWholeWordMatch and EvaluateMemoryRules are intentionally
// not exported — the redaction transform is the only operation callers
// outside this package need. Tests in this package access the helpers
// through the public RedactMemory entry point.
var _ = wholeWordSepRE // referenced via regexp init; keep variable to document
