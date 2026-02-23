// ABOUTME: Regex-based fast-path intent classifier using weighted keyword matching.
// ABOUTME: Returns Classification with intent, confidence, and contributing signals.

package intent

import (
	"regexp"
	"strings"
)

// keyword holds a pattern and its weight for scoring.
type keyword struct {
	pattern *regexp.Regexp
	word    string  // original keyword text for signal detail
	weight  float64 // contribution to intent score
}

// intentKeywords maps each intent to its weighted keyword patterns.
var intentKeywords map[Intent][]keyword

func init() {
	intentKeywords = map[Intent][]keyword{
		IntentPlan:     compileKeywords(planKeywords),
		IntentExecute:  compileKeywords(executeKeywords),
		IntentExplore:  compileKeywords(exploreKeywords),
		IntentDebug:    compileKeywords(debugKeywords),
		IntentRefactor: compileKeywords(refactorKeywords),
	}
}

// rawKeyword is an uncompiled keyword entry.
type rawKeyword struct {
	word   string
	weight float64
}

var planKeywords = []rawKeyword{
	{"plan", 1.0},
	{"design", 1.0},
	{"architect", 1.0},
	{"propose", 1.0},
	{"should we", 0.8},
	{"how should", 0.8},
	{"strategy", 0.9},
	{"approach", 0.8},
	{"structure", 0.7},
	{"organize", 0.7},
}

var executeKeywords = []rawKeyword{
	{"implement", 1.0},
	{"build", 0.9},
	{"create", 0.9},
	{"add", 0.7},
	{"write", 0.8},
	{"make", 0.7},
	{"generate", 0.8},
	{"set up", 0.8},
	{"install", 0.8},
	{"configure", 0.8},
	{"deploy", 0.9},
}

var exploreKeywords = []rawKeyword{
	{"explain", 1.0},
	{"show", 0.8},
	{"find", 0.8},
	{"search", 0.8},
	{"list", 0.7},
	{"what is", 0.9},
	{"where is", 0.9},
	{"how does", 0.9},
	{"read", 0.7},
	{"look at", 0.8},
	{"understand", 0.8},
}

var debugKeywords = []rawKeyword{
	{"fix", 1.0},
	{"bug", 1.0},
	{"error", 0.9},
	{"failing", 0.9},
	{"broken", 0.9},
	{"crash", 1.0},
	{"issue", 0.7},
	{"wrong", 0.8},
	{"not working", 1.0},
	{"debug", 1.0},
	{"diagnose", 0.9},
}

var refactorKeywords = []rawKeyword{
	{"refactor", 1.0},
	{"rename", 0.9},
	{"restructure", 0.9},
	{"clean up", 0.9},
	{"simplify", 0.8},
	{"extract", 0.8},
	{"move", 0.6},
	{"reorganize", 0.9},
	{"optimize", 0.8},
}

// compileKeywords turns raw keywords into compiled regex patterns with word boundaries.
// Multi-word phrases use exact boundaries; single words allow common suffixes.
func compileKeywords(raws []rawKeyword) []keyword {
	out := make([]keyword, len(raws))
	for i, rk := range raws {
		var pattern string
		if strings.Contains(rk.word, " ") {
			// Multi-word phrase: exact match with word boundaries.
			pattern = `(?i)\b` + regexp.QuoteMeta(rk.word) + `\b`
		} else {
			// Single word: allow common verb suffixes (e.g., crash -> crashes, crashing).
			pattern = `(?i)\b` + regexp.QuoteMeta(rk.word) + `(?:es|s|ed|ing)?\b`
		}
		out[i] = keyword{
			pattern: regexp.MustCompile(pattern),
			word:    rk.word,
			weight:  rk.weight,
		}
	}
	return out
}

// ClassifyHeuristic performs fast-path intent classification using keyword matching.
// It returns IntentAmbiguous with low confidence when no clear winner emerges.
func ClassifyHeuristic(input string) Classification {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return Classification{
			Intent:     IntentAmbiguous,
			Confidence: 0,
			Source:     "heuristic",
		}
	}

	type intentScore struct {
		intent  Intent
		score   float64
		signals []Signal
	}

	scores := make([]intentScore, 0, len(intentKeywords))

	for intent, keywords := range intentKeywords {
		var totalScore float64
		var signals []Signal

		for _, kw := range keywords {
			if kw.pattern.MatchString(trimmed) {
				totalScore += kw.weight
				signals = append(signals, Signal{
					Name:   "keyword_match",
					Weight: kw.weight,
					Detail: kw.word,
				})
			}
		}

		if totalScore > 0 {
			scores = append(scores, intentScore{
				intent:  intent,
				score:   totalScore,
				signals: signals,
			})
		}
	}

	if len(scores) == 0 {
		return Classification{
			Intent:     IntentAmbiguous,
			Confidence: 0,
			Source:     "heuristic",
		}
	}

	// Find the top two scores for tie detection.
	var best, secondBest intentScore
	for _, s := range scores {
		if s.score > best.score {
			secondBest = best
			best = s
		} else if s.score > secondBest.score {
			secondBest = s
		}
	}

	// If multiple intents scored and the gap is small, classify as ambiguous.
	if secondBest.score > 0 {
		ratio := secondBest.score / best.score
		if ratio >= 0.7 {
			// Merge signals from both for transparency.
			allSignals := make([]Signal, 0, len(best.signals)+len(secondBest.signals))
			allSignals = append(allSignals, best.signals...)
			allSignals = append(allSignals, secondBest.signals...)
			return Classification{
				Intent:     IntentAmbiguous,
				Confidence: 0.3,
				Source:     "heuristic",
				Signals:    allSignals,
			}
		}
	}

	// Compute confidence: normalize the winning score.
	// A single keyword with weight 1.0 gives ~0.5 confidence;
	// multiple matches push toward 1.0.
	confidence := normalizeConfidence(best.score)

	return Classification{
		Intent:     best.intent,
		Confidence: confidence,
		Source:     "heuristic",
		Signals:    best.signals,
	}
}

// normalizeConfidence maps a raw score to [0, 1] using a diminishing-returns curve.
// score=0.7 -> ~0.58, score=1.0 -> ~0.67, score=2.0 -> ~0.80, score=3.0 -> ~0.86
func normalizeConfidence(score float64) float64 {
	// f(x) = x / (x + k), where k controls the curve.
	const k = 0.35
	c := score / (score + k)
	if c > 1.0 {
		return 1.0
	}
	return c
}
