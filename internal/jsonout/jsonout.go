// Package jsonout writes the OpenDelve agent envelopes (decision + error) to
// stdout, and renders the same data as gh/kubectl-style human terminal output
// per devex F4. One source of truth for both audiences.
package jsonout

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// Decision is the success envelope (matches schemas/decision.schema.json).
type Decision struct {
	SchemaVersion     string                 `json:"schemaVersion"`
	Status            string                 `json:"status"`
	Subject           string                 `json:"subject"`
	SubjectType       string                 `json:"subjectType,omitempty"`
	PolicyRefs        []string               `json:"policyRefs,omitempty"`
	RequiredEvidence  []string               `json:"requiredEvidence,omitempty"`
	MissingEvidence   []string               `json:"missingEvidence,omitempty"`
	RequiredApprovals []string               `json:"requiredApprovals,omitempty"`
	MissingApprovals  []string               `json:"missingApprovals,omitempty"`
	BlockedActions    []string               `json:"blockedActions,omitempty"`
	AllowedNext       []string               `json:"allowedNext,omitempty"`
	Git               *GitContext            `json:"git,omitempty"`
	Data              any                    `json:"data,omitempty"`
	Timestamp         string                 `json:"timestamp"`
	extra             map[string]any         `json:"-"`
}

// Decision status enums (matches the schema).
const (
	StatusOK             = "ok"
	StatusNeedsApproval  = "needs_approval"
	StatusNeedsEvidence  = "needs_evidence"
	StatusBlocked        = "blocked"
)

// Error is the error envelope (matches schemas/error.schema.json).
// Distinct shape from Decision; agents discriminate on the top-level "kind" field.
type Error struct {
	SchemaVersion string         `json:"schemaVersion"`
	Kind          string         `json:"kind"` // always "error"
	Code          string         `json:"code"`
	Message       string         `json:"message"`
	Hint          string         `json:"hint,omitempty"`
	DocURL        string         `json:"docUrl,omitempty"`
	Details       map[string]any `json:"details,omitempty"`
	Subject       string         `json:"subject,omitempty"`
	Timestamp     string         `json:"timestamp"`
}

// GitContext is the embedded git state in a Decision envelope.
type GitContext struct {
	Commit string `json:"commit,omitempty"`
	Branch string `json:"branch,omitempty"`
	Dirty  bool   `json:"dirty,omitempty"`
}

// NewDecision returns a fresh Decision envelope with schemaVersion+timestamp set.
func NewDecision(status, subject string) *Decision {
	return &Decision{
		SchemaVersion: "v1",
		Status:        status,
		Subject:       subject,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}
}

// NewError returns a fresh Error envelope with schemaVersion+kind+timestamp set.
func NewError(code, message string) *Error {
	return &Error{
		SchemaVersion: "v1",
		Kind:          "error",
		Code:          code,
		Message:       message,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}
}

// WithHint sets the hint string.
func (e *Error) WithHint(hint string) *Error { e.Hint = hint; return e }

// WithSubject sets the subject id.
func (e *Error) WithSubject(s string) *Error { e.Subject = s; return e }

// WithDocURL sets the docs link.
func (e *Error) WithDocURL(u string) *Error { e.DocURL = u; return e }

// WithDetails sets the details map.
func (e *Error) WithDetails(d map[string]any) *Error { e.Details = d; return e }

// WriteJSON marshals an envelope to w as indented JSON followed by a newline.
// HTML escaping is disabled so '<', '>', '&' render literally — agents reading
// the envelope expect raw characters in fields like 'allowedNext' and hints.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// Mode describes how output should render. Default is JSONIfNonTTY.
type Mode int

const (
	// ModeAuto picks JSON if stdout is not a TTY, otherwise human.
	ModeAuto Mode = iota
	// ModeJSON forces JSON output regardless of TTY.
	ModeJSON
	// ModeHuman forces human (colored) output regardless of TTY.
	ModeHuman
)

// Emit writes a Decision or Error envelope according to mode + stdoutIsTTY.
// Returns the appropriate exit code per eng review C1:
//
//	0 — Decision with status "ok"
//	2 — Decision with status "needs_*" or "blocked"
//	1 — Error envelope
func Emit(stdout, stderr io.Writer, stdoutIsTTY bool, mode Mode, v any) int {
	useJSON := mode == ModeJSON || (mode == ModeAuto && !stdoutIsTTY)

	switch env := v.(type) {
	case *Decision:
		if useJSON {
			_ = WriteJSON(stdout, env)
		} else {
			renderDecisionHuman(stdout, env)
		}
		switch env.Status {
		case StatusOK:
			return 0
		default:
			return 2
		}
	case *Error:
		if useJSON {
			_ = WriteJSON(stdout, env)
		} else {
			renderErrorHuman(stderr, env)
		}
		return 1
	default:
		// Programmer error.
		fmt.Fprintf(stderr, "internal: jsonout.Emit got unexpected type %T\n", v)
		return 1
	}
}

// renderDecisionHuman prints a Decision in human-friendly terminal form.
func renderDecisionHuman(w io.Writer, d *Decision) {
	switch d.Status {
	case StatusOK:
		fmt.Fprintf(w, "%s ok: %s\n", okPrefix(), d.Subject)
	case StatusNeedsApproval:
		fmt.Fprintf(w, "%s needs approval: %s\n", warnPrefix(), d.Subject)
		if len(d.MissingApprovals) > 0 {
			fmt.Fprintf(w, "  missing: %v\n", d.MissingApprovals)
		}
	case StatusNeedsEvidence:
		fmt.Fprintf(w, "%s needs evidence: %s\n", warnPrefix(), d.Subject)
		if len(d.MissingEvidence) > 0 {
			fmt.Fprintf(w, "  missing: %v\n", d.MissingEvidence)
		}
	case StatusBlocked:
		fmt.Fprintf(w, "%s blocked: %s\n", errPrefix(), d.Subject)
		if len(d.BlockedActions) > 0 {
			fmt.Fprintf(w, "  blocked actions: %v\n", d.BlockedActions)
		}
	default:
		fmt.Fprintf(w, "%s status %q: %s\n", warnPrefix(), d.Status, d.Subject)
	}
	if len(d.AllowedNext) > 0 {
		fmt.Fprintf(w, "  next: %v\n", d.AllowedNext)
	}
}

// renderErrorHuman prints an Error in gh/kubectl three-line form per devex F4.
func renderErrorHuman(w io.Writer, e *Error) {
	fmt.Fprintf(w, "%s %s\n", errPrefix(), e.Message)
	if e.Hint != "" {
		fmt.Fprintf(w, "%s %s\n", hintPrefix(), e.Hint)
	}
	if e.DocURL != "" {
		fmt.Fprintf(w, "%s %s\n", learnPrefix(), e.DocURL)
	}
}

// IsTTY reports whether the given file descriptor is a TTY. Returns false on
// any error (treats unknown as non-TTY for safety in CI).
func IsTTY(f *os.File) bool {
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return (st.Mode() & os.ModeCharDevice) != 0
}

// Color/prefix helpers. ANSI escape codes are emitted only when stdout is a
// TTY in human mode; the renderer functions here always emit them and the
// terminal does (or doesn't) interpret them. A future patch can add a
// no-color env var check.

func okPrefix() string   { return "\033[32m✓\033[0m" }    // green check
func warnPrefix() string { return "\033[33m!\033[0m" }    // yellow bang
func errPrefix() string  { return "\033[31merror:\033[0m" } // red error:
func hintPrefix() string { return "\033[2mhint:\033[0m" }   // dim hint:
func learnPrefix() string {
	return "\033[2mlearn more:\033[0m" // dim learn more:
}
