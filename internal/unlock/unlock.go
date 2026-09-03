package unlock

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/oneclickvirt/UnlockTests/executor"
)

type Result struct {
	Name       string
	Status     string
	Region     string
	Info       string
	UnlockType string
	Error      string
	Raw        map[string]any
}

type Runner struct {
	selection   string
	ipVersion   string
	language    string
	cache       bool
	concurrency int
}

func New(selection, ipVersion string) *Runner {
	return &Runner{selection: selection, ipVersion: ipVersion, language: "zh", concurrency: 20}
}

func (r *Runner) WithLanguage(language string) *Runner {
	r.language = language
	return r
}

func (r *Runner) WithCache(enabled bool) *Runner {
	r.cache = enabled
	return r
}

func (r *Runner) WithConcurrency(n int) *Runner {
	if n > 0 {
		r.concurrency = n
	}
	return r
}

func (r *Runner) Results(ctx context.Context) ([]executor.StructuredResult, error) {
	results, err := executor.RunStructured(ctx, executor.RunOptions{
		Selection:   r.selection,
		IPVersion:   r.ipVersion,
		Concurrency: r.concurrency,
		UseCache:    r.cache,
	})
	if err != nil && len(results) == 0 {
		return nil, err
	}
	return results, err
}

func (r *Runner) UnlockedPlatforms(ctx context.Context) ([]string, error) {
	results, err := r.Results(ctx)
	if err != nil && len(results) == 0 {
		return nil, err
	}
	return collectByStatus(results, func(status string) bool {
		return strings.EqualFold(status, "YES") || strings.EqualFold(status, "CDN Relay Available")
	}), err
}

func (r *Runner) LockedPlatforms(ctx context.Context) ([]string, error) {
	results, err := r.Results(ctx)
	if err != nil && len(results) == 0 {
		return nil, err
	}
	return collectByStatus(results, func(status string) bool {
		return !(strings.EqualFold(status, "YES") || strings.EqualFold(status, "CDN Relay Available"))
	}), err
}

func (r *Runner) ListPlatforms(ctx context.Context) ([]string, error) {
	_ = ctx
	platforms, err := executor.ListPlatforms(r.selection)
	if err != nil {
		return nil, fmt.Errorf("list platforms: %w", err)
	}
	return platforms, nil
}

func collectByStatus(results []executor.StructuredResult, match func(string) bool) []string {
	platforms := make([]string, 0, len(results))
	seen := make(map[string]struct{}, len(results))
	for _, item := range results {
		name := strings.TrimSpace(item.Name)
		if name == "" || !match(item.Status) {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		platforms = append(platforms, name)
	}
	sort.Strings(platforms)
	return platforms
}
