// ABOUTME: Thin wrapper over sahilm/fuzzy for fuzzy string matching
// ABOUTME: Provides a simplified API for filtering and ranking matches

package fuzzy

import "github.com/sahilm/fuzzy"

// Match represents a single fuzzy match result.
type Match struct {
	Str            string
	Index          int
	MatchedIndexes []int
	Score          int
}

// Find performs fuzzy matching of pattern against the given items.
// Returns matches sorted by score (best first).
func Find(pattern string, items []string) []Match {
	results := fuzzy.Find(pattern, items)
	matches := make([]Match, len(results))
	for i, r := range results {
		matches[i] = Match{
			Str:            r.Str,
			Index:          r.Index,
			MatchedIndexes: r.MatchedIndexes,
			Score:          r.Score,
		}
	}
	return matches
}

// FindFrom performs fuzzy matching using a custom string source.
func FindFrom(pattern string, data fuzzy.Source) []Match {
	results := fuzzy.FindFrom(pattern, data)
	matches := make([]Match, len(results))
	for i, r := range results {
		matches[i] = Match{
			Str:            r.Str,
			Index:          r.Index,
			MatchedIndexes: r.MatchedIndexes,
			Score:          r.Score,
		}
	}
	return matches
}
