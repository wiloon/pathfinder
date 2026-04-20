# Task Spec: #18 — ParseGoalText AI Interface

**Status:** Pending
**Priority:** P2
**Depends on:** #23 (Goal model refactor ✅)
**Related ADR:** [ADR-0003](../adr/0003-ai-provider-architecture.md), [ADR-0007](../adr/0007-ux-patterns.md)

---

## Summary

Add `ParseGoalText(rawInput string) (ParsedGoal, error)` to the `ai.Provider` interface. Implement in `MiniMaxProvider` using JSON-mode prompting. `OpenClawProvider` returns a minimal deterministic fallback. `ParsedGoal` is a new exported struct in the `ai/` package.

---

## ParsedGoal Struct

```go
// ParsedGoal is the structured output of AI goal text parsing.
// All fields are populated by the AI; the caller may edit before persisting.
type ParsedGoal struct {
    Title       string   // Concise goal title extracted from rawInput
    Description string   // Elaborated description
    Weight      int      // 1–10; AI-estimated importance; clamped to 5 if uncertain or out of range
    Tags        []string // Category tags, e.g. ["career", "health"]; empty slice if none detected
    Timeline    string   // e.g. "3 months"; empty string if no deadline mentioned (long-term)
}
```

---

## Provider Interface Addition

```go
// ParseGoalText parses a free-text goal description into a structured ParsedGoal.
// rawInput must be pre-validated (non-empty, max 500 chars) by the caller.
ParseGoalText(rawInput string) (ParsedGoal, error)
```

Add to the `Provider` interface in `ai/ai.go`, and add the package-level dispatcher:

```go
func ParseGoalText(rawInput string) (ParsedGoal, error) {
    return activeProvider.ParseGoalText(rawInput)
}
```

---

## MiniMaxProvider Implementation

**Model call:** Use existing `ChatCompletion` infrastructure. Set `response_format` to JSON mode if supported, or instruct via system prompt.

**System message (exact text):**
```
You are a goal-setting assistant. Parse the user's goal description and return a JSON object with exactly these fields:
- "title": string, concise, max 80 chars
- "description": string, elaborated, max 300 chars
- "weight": integer 1-10 (1=lowest priority, 10=highest; estimate from urgency/importance language in the description; use 5 if unclear)
- "tags": array of strings, choose from: career, health, education, personal, finance, relationships, other; empty array if none apply
- "timeline": string, e.g. "3 months", "6 weeks"; empty string if no deadline mentioned

Respond with valid JSON only. No markdown, no explanation, no code fences.
```

**User message:** `json.Marshal(rawInput)` — rawInput is JSON-encoded as a string value (prevents prompt injection; see C4).

**Response parsing:**
- Unmarshal response content into `ParsedGoal`
- If `Weight` < 1 or > 10: clamp to 5
- If `Title` is empty after unmarshal: return `fmt.Errorf("parseGoalText: AI returned empty title")`
- If JSON unmarshal fails: return `fmt.Errorf("parseGoalText: %w (raw: %.200s)", err, rawResponse)`

---

## OpenClawProvider Fallback

OpenClaw users reach goal parsing via the full brief flow (`POST /api/plan/brief`), not this endpoint. Return a minimal fallback so the interface is satisfied:

```go
func (p *OpenClawProvider) ParseGoalText(rawInput string) (ai.ParsedGoal, error) {
    return ai.ParsedGoal{
        Title:       rawInput,
        Description: "",
        Weight:      5,
        Tags:        []string{},
        Timeline:    "",
    }, nil
}
```

---

## Tests

File: `ai/ai_test.go` (add to existing test file). Mock the MiniMax HTTP endpoint using `httptest.NewServer`. 5 test cases:

1. **Happy path** — mock returns valid JSON with all fields; assert `ParsedGoal` fields match expected values
2. **Weight out of range** — mock returns `{"title":"X","weight":99,...}`; assert `Weight` clamped to 5
3. **Empty title** — mock returns `{"title":"","weight":5,...}`; assert error returned
4. **Invalid JSON** — mock returns `"this is not json"`; assert error returned (error message contains raw snippet)
5. **OpenClaw fallback** — init provider as OpenClaw; call `ParseGoalText("some input")`; assert `Title == "some input"`, `Weight == 5`, `Tags` empty, no error

---

## Out of Scope

- The HTTP handler that calls this function (task #19)
- Caching parsed results
- Multi-language input handling beyond what MiniMax natively supports
- Retry logic on MiniMax timeout (caller handles errors)
