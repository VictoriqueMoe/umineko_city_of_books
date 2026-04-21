// Package contentfilter is the single choke point all services call before
// persisting user-supplied text. New validation rules plug in as Rule
// implementations without changing callers.
package contentfilter

import "context"

type RuleName string

const (
	RuleBannedGiphy RuleName = "banned_giphy"
	RuleSlurs       RuleName = "slurs"
)

type (
	Rejection struct {
		Rule   RuleName `json:"rule"`
		Reason string   `json:"reason"`
		Detail string   `json:"detail,omitempty"`
	}

	Rule interface {
		Name() RuleName
		Check(ctx context.Context, texts []string) (*Rejection, error)
	}

	RejectedError struct {
		Rejection Rejection
	}

	// Manager holds the registered rules and runs them in order. Services hold
	// a *Manager and call Check before persisting any user-supplied text.
	Manager struct {
		rules []Rule
	}
)

func (e *RejectedError) Error() string {
	return e.Rejection.Reason
}

func New(rules ...Rule) *Manager {
	return &Manager{rules: rules}
}

// Check runs every rule in order over the supplied text fields. Empty strings
// are skipped. On the first rejection the Manager short-circuits and returns a
// *RejectedError. Infrastructure errors from a rule are returned as plain errors.
func (m *Manager) Check(ctx context.Context, texts ...string) error {
	filtered := make([]string, 0, len(texts))
	for _, t := range texts {
		if t != "" {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	for _, r := range m.rules {
		rej, err := r.Check(ctx, filtered)
		if err != nil {
			return err
		}
		if rej != nil {
			return &RejectedError{Rejection: *rej}
		}
	}
	return nil
}
