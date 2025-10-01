package search

import (
	"sort"
	"strings"

	"github.com/SagenKoder/launcher/internal/applications"
)

// Filter returns the subset of applications that match the query using a simple
// fuzzy subsequence match. Results are ordered by match quality and name.
func Filter(apps []applications.Application, query string) []applications.Application {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil
	}

	q := strings.ToLower(trimmed)
	results := scoreApplications(apps, q)
	if len(results) == 0 {
		return nil
	}

	filtered := make([]applications.Application, len(results))
	for i, res := range results {
		filtered[i] = res.app
	}
	return filtered
}

type scoredApp struct {
	app   applications.Application
	score int
	kind  string
}

func scoreApplications(apps []applications.Application, q string) []scoredApp {
	results := make([]scoredApp, 0, 32)
	for _, app := range apps {
		if score, kind := scoreApplication(app, q); score > 0 {
			results = append(results, scoredApp{app: app, score: score, kind: kind})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].score == results[j].score {
			nameI := strings.ToLower(results[i].app.Name)
			nameJ := strings.ToLower(results[j].app.Name)
			if nameI == nameJ {
				return results[i].app.Name < results[j].app.Name
			}
			return nameI < nameJ
		}
		return results[i].score > results[j].score
	})
	return results
}

func scoreApplication(app applications.Application, q string) (int, string) {
	lowerName := strings.ToLower(app.Name)
	lowerExec := strings.ToLower(app.Exec)

	if idx := strings.Index(lowerName, q); idx >= 0 {
		score := 2000 - idx*20 - len(lowerName)
		return score, "name-substring"
	}
	if idx := strings.Index(lowerExec, q); idx >= 0 {
		score := 1500 - idx*20 - len(lowerExec)
		return score, "exec-substring"
	}

	if ok, score := fuzzyScore(q, lowerName); ok {
		return 1000 + score, "name-fuzzy"
	}
	if ok, score := fuzzyScore(q, lowerExec); ok {
		return 800 + score, "exec-fuzzy"
	}
	return 0, ""
}

func fuzzyScore(query, candidate string) (bool, int) {
	qr := []rune(query)
	cr := []rune(candidate)
	if len(qr) == 0 || len(cr) == 0 {
		return false, 0
	}

	qi := 0
	score := 0
	lastMatchIndex := -1

	for ci, r := range cr {
		if qi >= len(qr) {
			break
		}
		if r == qr[qi] {
			score += 5
			if lastMatchIndex == ci-1 {
				score += 10 // reward consecutive matches
			}
			if qi == 0 {
				score += max(15-ci*2, 0) // prefer matches near the start
			}
			lastMatchIndex = ci
			qi++
		}
	}

	if qi != len(qr) {
		return false, 0
	}

	return true, score - len(cr)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// DebugScore exposes how an application scored for a given query.
// Intended for diagnostics only.
func DebugScore(app applications.Application, query string) (string, int) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return "", 0
	}
	score, kind := scoreApplication(app, strings.ToLower(trimmed))
	return kind, score
}
