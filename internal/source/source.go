package source

import "strings"

// Target represents a connectable remote target from any source.
type Target struct {
	Source   string            // "ssh", "gce", "gke"
	Alias    string            // display name
	Group    string            // grouping key
	Detail   string            // one-line description (e.g. user@host, zone, namespace)
	Meta     map[string]string // source-specific metadata
	Editable bool
}

// Source provides a list of connectable targets.
type Source interface {
	Name() string
	Fetch() ([]Target, error)
	Connect(t Target) error
}

// GroupTargets organizes targets by group, preserving first-seen order.
func GroupTargets(targets []Target) []TargetGroup {
	var order []string
	grouped := make(map[string][]Target)
	for _, t := range targets {
		g := t.Group
		if g == "" {
			g = "ungrouped"
		}
		if _, exists := grouped[g]; !exists {
			order = append(order, g)
		}
		grouped[g] = append(grouped[g], t)
	}

	result := make([]TargetGroup, len(order))
	for i, g := range order {
		result[i] = TargetGroup{Name: g, Targets: grouped[g]}
	}
	return result
}

// FilterTargets returns targets matching the query on alias, group, or detail.
func FilterTargets(targets []Target, query string) []Target {
	if query == "" {
		return targets
	}
	q := strings.ToLower(query)
	var result []Target
	for _, t := range targets {
		if strings.Contains(strings.ToLower(t.Alias), q) ||
			strings.Contains(strings.ToLower(t.Group), q) ||
			strings.Contains(strings.ToLower(t.Detail), q) {
			result = append(result, t)
		}
	}
	return result
}

type TargetGroup struct {
	Name    string
	Targets []Target
}
