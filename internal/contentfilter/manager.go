package contentfilter

import "context"

type RuleName string

const (
	RuleBannedGiphy     RuleName = "banned_giphy"
	RuleSlurs           RuleName = "slurs"
	RuleNewAccountLinks RuleName = "new_account_links"
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

func (m *Manager) Check(ctx context.Context, texts ...string) error {
	if m == nil {
		return nil
	}

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
