// ABOUTME: Table-driven tests for heuristic-based intent classification.
// ABOUTME: Covers all intent types, edge cases, ambiguous inputs, and confidence thresholds.

package intent

import (
	"strings"
	"testing"
)

func TestClassifyHeuristic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		wantIntent    Intent
		minConfidence float64
	}{
		// ── Plan intent (20 cases) ──────────────────────────────────────
		{"plan: design a new API", "design a new API for user management", IntentPlan, 0.6},
		{"plan: plan the architecture", "plan the architecture for the payment service", IntentPlan, 0.6},
		{"plan: architect the system", "architect the system for high availability", IntentPlan, 0.6},
		{"plan: propose a solution", "propose a solution for the caching problem", IntentPlan, 0.6},
		{"plan: should we use microservices", "should we use microservices or monolith", IntentPlan, 0.6},
		{"plan: how should we structure", "how should we structure the database layer", IntentPlan, 0.6},
		{"plan: strategy for migration", "the strategy for database migration", IntentPlan, 0.6},
		{"plan: approach for testing", "what approach should we take for testing", IntentPlan, 0.6},
		{"plan: structure the project", "structure the project into modules", IntentPlan, 0.6},
		{"plan: organize the codebase", "organize the codebase by domain", IntentPlan, 0.6},
		{"plan: design pattern question", "design a pattern for event sourcing", IntentPlan, 0.6},
		{"plan: plan out the feature", "plan out the authentication feature", IntentPlan, 0.6},
		{"plan: propose an architecture", "propose an architecture for the new service", IntentPlan, 0.6},
		{"plan: how should we approach", "how should we approach error handling", IntentPlan, 0.6},
		{"plan: strategy for scaling", "strategy for scaling the notification system", IntentPlan, 0.6},
		{"plan: architect a pipeline", "architect a CI/CD pipeline", IntentPlan, 0.6},
		{"plan: design the schema", "design the database schema for orders", IntentPlan, 0.6},
		{"plan: plan the deployment", "plan the deployment strategy for Kubernetes", IntentPlan, 0.6},
		{"plan: structure for modularity", "structure the code for better modularity", IntentPlan, 0.6},
		{"plan: organize imports", "organize the service layer around domains", IntentPlan, 0.6},

		// ── Execute intent (20 cases) ───────────────────────────────────
		{"exec: implement the handler", "implement the HTTP handler for /users", IntentExecute, 0.6},
		{"exec: build the CLI", "build the CLI tool for deployment", IntentExecute, 0.6},
		{"exec: create a new file", "create a new file called server.go", IntentExecute, 0.6},
		{"exec: add a test", "add a test for the parser function", IntentExecute, 0.6},
		{"exec: write a function", "write a function to validate email addresses", IntentExecute, 0.6},
		{"exec: make a middleware", "make a middleware for rate limiting", IntentExecute, 0.6},
		{"exec: generate the code", "generate the code for the protobuf schema", IntentExecute, 0.6},
		{"exec: set up the project", "set up the project with Go modules", IntentExecute, 0.6},
		{"exec: install the dependency", "install the dependency for JSON parsing", IntentExecute, 0.6},
		{"exec: configure the linter", "configure the linter for the project", IntentExecute, 0.6},
		{"exec: deploy to staging", "deploy the service to staging", IntentExecute, 0.6},
		{"exec: implement auth", "implement authentication with JWT tokens", IntentExecute, 0.6},
		{"exec: build the docker image", "build the docker image for production", IntentExecute, 0.6},
		{"exec: create a migration", "create a database migration for the users table", IntentExecute, 0.6},
		{"exec: add validation", "add input validation to the API endpoints", IntentExecute, 0.6},
		{"exec: write the integration test", "write the integration test for checkout flow", IntentExecute, 0.6},
		{"exec: make it concurrent", "make the data processor concurrent", IntentExecute, 0.6},
		{"exec: generate mocks", "generate mocks for the repository interface", IntentExecute, 0.6},
		{"exec: set up CI", "set up CI pipeline with GitHub Actions", IntentExecute, 0.6},
		{"exec: configure logging", "configure the logging output format for the service", IntentExecute, 0.6},

		// ── Explore intent (20 cases) ───────────────────────────────────
		{"explore: explain the code", "explain the code in the handler package", IntentExplore, 0.6},
		{"explore: show me the config", "show me the configuration options", IntentExplore, 0.6},
		{"explore: find the function", "find the function that handles authentication", IntentExplore, 0.6},
		{"explore: search for usage", "search for usage of the deprecated API", IntentExplore, 0.6},
		{"explore: list all endpoints", "list all endpoints in the API", IntentExplore, 0.6},
		{"explore: what is this type", "what is the purpose of this type", IntentExplore, 0.6},
		{"explore: where is the config", "where is the configuration loaded from", IntentExplore, 0.6},
		{"explore: how does auth work", "how does the authentication middleware work", IntentExplore, 0.6},
		{"explore: read the file", "read the file internal/config/settings.go", IntentExplore, 0.6},
		{"explore: look at the tests", "look at the tests in the permission package", IntentExplore, 0.6},
		{"explore: understand the flow", "understand the request flow through the server", IntentExplore, 0.6},
		{"explore: explain error handling", "explain how the middleware pipeline works in this project", IntentExplore, 0.6},
		{"explore: show the schema", "show me the database schema definition", IntentExplore, 0.6},
		{"explore: find all imports", "find all imports of the eventbus package", IntentExplore, 0.6},
		{"explore: what is the purpose", "what is the purpose of the sandbox module", IntentExplore, 0.6},
		{"explore: where is it defined", "where is the Config struct defined", IntentExplore, 0.6},
		{"explore: how does caching work", "how does the caching layer work", IntentExplore, 0.6},
		{"explore: list the dependencies", "list the external dependencies", IntentExplore, 0.6},
		{"explore: search for pattern", "search for the observer pattern implementation", IntentExplore, 0.6},
		{"explore: look at the logs", "look at the logging implementation", IntentExplore, 0.6},

		// ── Debug intent (20 cases) ─────────────────────────────────────
		{"debug: fix the test", "fix the failing test in the handler package", IntentDebug, 0.6},
		{"debug: bug in parser", "there is a bug in the parser for nested objects", IntentDebug, 0.6},
		{"debug: error in build", "I get an error when running the project", IntentDebug, 0.6},
		{"debug: test failing", "the test TestParseConfig is failing", IntentDebug, 0.6},
		{"debug: broken endpoint", "the /users endpoint is broken", IntentDebug, 0.6},
		{"debug: crash on startup", "the server crashes on startup with nil pointer", IntentDebug, 0.6},
		{"debug: issue with auth", "there is an issue with the authentication flow", IntentDebug, 0.6},
		{"debug: wrong output", "the function returns the wrong output for edge cases", IntentDebug, 0.6},
		{"debug: not working", "the webhook handler is not working correctly", IntentDebug, 0.6},
		{"debug: debug the query", "debug the SQL query that returns empty results", IntentDebug, 0.6},
		{"debug: diagnose the leak", "diagnose the memory leak in the cache", IntentDebug, 0.6},
		{"debug: fix nil pointer", "fix the nil pointer dereference in config loading", IntentDebug, 0.6},
		{"debug: error handling broken", "error handling is broken in the middleware", IntentDebug, 0.6},
		{"debug: test suite failing", "the entire test suite is failing after the merge", IntentDebug, 0.6},
		{"debug: broken migration", "the database migration is broken", IntentDebug, 0.6},
		{"debug: crash with large input", "the service crashes with large input payloads", IntentDebug, 0.6},
		{"debug: issue with concurrency", "there is a race condition issue with the cache", IntentDebug, 0.6},
		{"debug: wrong status code", "the API returns wrong status codes for errors", IntentDebug, 0.6},
		{"debug: not working after update", "the parser is not working after the last update", IntentDebug, 0.6},
		{"debug: diagnose timeout", "diagnose why requests are timing out", IntentDebug, 0.6},

		// ── Refactor intent (20 cases) ──────────────────────────────────
		{"refactor: refactor the handler", "refactor the handler to use dependency injection", IntentRefactor, 0.6},
		{"refactor: rename the variable", "rename the variable from x to userCount", IntentRefactor, 0.6},
		{"refactor: restructure the package", "restructure the package layout for clarity", IntentRefactor, 0.6},
		{"refactor: clean up the code", "clean up the code in the service layer", IntentRefactor, 0.6},
		{"refactor: simplify the logic", "simplify the logic in the validation function", IntentRefactor, 0.6},
		{"refactor: extract a function", "extract the retry logic into a separate function", IntentRefactor, 0.6},
		{"refactor: move the file", "move the types file to the domain package", IntentRefactor, 0.6},
		{"refactor: reorganize the tests", "reorganize the test files by feature", IntentRefactor, 0.6},
		{"refactor: optimize the query", "optimize the database query for better performance", IntentRefactor, 0.6},
		{"refactor: refactor to interfaces", "refactor the concrete types to use interfaces", IntentRefactor, 0.6},
		{"refactor: rename the package", "rename the package from utils to helpers", IntentRefactor, 0.6},
		{"refactor: restructure the API", "restructure the API routes for versioning", IntentRefactor, 0.6},
		{"refactor: clean up imports", "clean up unused imports across the project", IntentRefactor, 0.6},
		{"refactor: simplify validation", "simplify the validation chain in the parser", IntentRefactor, 0.6},
		{"refactor: extract middleware", "extract common middleware into a shared package", IntentRefactor, 0.6},
		{"refactor: move to internal", "move the public types to internal package", IntentRefactor, 0.6},
		{"refactor: reorganize config", "reorganize the config loading into smaller files", IntentRefactor, 0.6},
		{"refactor: optimize the loop", "optimize the loop that processes events", IntentRefactor, 0.6},
		{"refactor: refactor the tests", "refactor the tests to use table-driven style", IntentRefactor, 0.6},
		{"refactor: extract interface", "extract an interface from the concrete repository", IntentRefactor, 0.6},

		// ── Ambiguous / mixed intent ────────────────────────────────────
		{"ambiguous: fix then refactor", "refactor the code and fix it", IntentAmbiguous, 0.0},
		{"ambiguous: hello", "hello", IntentAmbiguous, 0.0},
		{"ambiguous: thanks", "thanks", IntentAmbiguous, 0.0},
		{"ambiguous: ok", "ok", IntentAmbiguous, 0.0},
		{"ambiguous: plan and implement", "plan the feature and implement it", IntentAmbiguous, 0.0},
		{"ambiguous: generic question", "what do you think about this", IntentAmbiguous, 0.0},

		// ── Edge cases ──────────────────────────────────────────────────
		{"edge: empty string", "", IntentAmbiguous, 0.0},
		{"edge: single word plan", "plan", IntentPlan, 0.3},
		{"edge: single word fix", "fix", IntentDebug, 0.3},
		{"edge: whitespace only", "   ", IntentAmbiguous, 0.0},
		{"edge: very long input plan", "design " + strings.Repeat("a detailed system ", 100) + "for scalability", IntentPlan, 0.3},
		{"edge: case insensitive", "IMPLEMENT the new feature NOW", IntentExecute, 0.6},
		{"edge: mixed case", "FiX the BuG in the PaRsEr", IntentDebug, 0.6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ClassifyHeuristic(tt.input)

			if got.Intent != tt.wantIntent {
				t.Errorf("ClassifyHeuristic(%q).Intent = %v; want %v (signals: %v)",
					tt.input, got.Intent, tt.wantIntent, got.Signals)
			}

			if got.Confidence < tt.minConfidence {
				t.Errorf("ClassifyHeuristic(%q).Confidence = %.2f; want >= %.2f",
					tt.input, got.Confidence, tt.minConfidence)
			}

			if got.Source != "heuristic" {
				t.Errorf("ClassifyHeuristic(%q).Source = %q; want %q",
					tt.input, got.Source, "heuristic")
			}

			if got.Confidence < 0 || got.Confidence > 1 {
				t.Errorf("ClassifyHeuristic(%q).Confidence = %.2f; out of [0,1] range",
					tt.input, got.Confidence)
			}
		})
	}
}

func TestClassifyHeuristic_SignalsPopulated(t *testing.T) {
	t.Parallel()

	got := ClassifyHeuristic("implement the HTTP handler for users")
	if len(got.Signals) == 0 {
		t.Error("expected non-empty Signals for a clear intent")
	}

	foundKeyword := false
	for _, s := range got.Signals {
		if s.Name == "keyword_match" {
			foundKeyword = true
			if s.Detail == "" {
				t.Error("keyword_match signal should have a non-empty Detail")
			}
			if s.Weight <= 0 {
				t.Errorf("keyword_match signal weight = %.2f; want > 0", s.Weight)
			}
		}
	}
	if !foundKeyword {
		t.Error("expected at least one keyword_match signal")
	}
}

func TestClassifyHeuristic_EmptyReturnsSane(t *testing.T) {
	t.Parallel()

	got := ClassifyHeuristic("")
	if got.Intent != IntentAmbiguous {
		t.Errorf("empty input: got %v; want IntentAmbiguous", got.Intent)
	}
	if got.Confidence != 0 {
		t.Errorf("empty input: confidence = %.2f; want 0", got.Confidence)
	}
}

func TestIntent_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		intent Intent
		want   string
	}{
		{IntentPlan, "plan"},
		{IntentExecute, "execute"},
		{IntentExplore, "explore"},
		{IntentDebug, "debug"},
		{IntentRefactor, "refactor"},
		{IntentAmbiguous, "ambiguous"},
		{Intent(99), "unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.intent.String(); got != tt.want {
				t.Errorf("Intent(%d).String() = %q; want %q", tt.intent, got, tt.want)
			}
		})
	}
}
