package ai

import (
	"pathfinder-api/storage"
)

// ParsedEvent is a concrete near-term event extracted from free-form goal text.
type ParsedEvent struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	EventDate   string `json:"event_date"` // YYYY-MM-DD; empty when date is ambiguous
	Note        string `json:"note"`       // original time reference from the input
}

// ParsedTask is an immediate preparation task that must start today (or a specified date).
type ParsedTask struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Date        string `json:"date"` // YYYY-MM-DD; defaults to today when empty
}

// ParsedGoal is the structured result of AI goal parsing.
type ParsedGoal struct {
	Title           string       `json:"title"`
	Description     string       `json:"description"`
	Weight          int          `json:"weight"` // 1–10; clamped to range
	Tags            []string     `json:"tags"`
	Timeline        string       `json:"timeline"`         // empty = long-term
	ExtractedEvents []ParsedEvent `json:"extracted_events"` // calendar events with specific future dates
	ExtractedTasks  []ParsedTask  `json:"extracted_tasks"`  // prep work that must start today
}

// Provider is the interface every AI backend must implement.
type Provider interface {
	GenerateInitialPlan(goals []storage.Goal, attachments []storage.GoalAttachment, availableHours float64, startTime string) ([]storage.Task, error)
	RegenerateAfterCheckin(checkin storage.CheckIn, recentHistory []storage.DailyPlan, upcomingEvents []storage.Event) ([]storage.Task, error)
	InsertEvent(event storage.Event, upcomingPlans []storage.DailyPlan) ([]storage.Task, error)
	ParseGoalText(rawInput string) (ParsedGoal, error)
}

// Config holds configuration for all AI providers.
type Config struct {
	// Provider selects the active backend: "minimax" (default) or "openclaw".
	Provider string

	// MiniMax fields
	APIKey  string
	Model   string
	BaseURL string

	// OpenClaw fields
	OpenClawSyncURL       string
	OpenClawWebhookURL    string
	OpenClawWebhookSecret string
	OpenClawServiceToken  string // Pathfinder → OpenClaw auth token
}

var activeProvider Provider

// Init sets the active provider based on cfg.Provider.
// Must be called once at startup before any package-level functions are used.
func Init(cfg Config) {
	switch cfg.Provider {
	case "openclaw":
		activeProvider = &OpenClawProvider{
			SyncURL:       cfg.OpenClawSyncURL,
			WebhookURL:    cfg.OpenClawWebhookURL,
			WebhookSecret: cfg.OpenClawWebhookSecret,
			ServiceToken:  cfg.OpenClawServiceToken,
		}
	default: // "minimax" or unset
		activeProvider = &MiniMaxProvider{
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			BaseURL: cfg.BaseURL,
		}
	}
}

// GenerateInitialPlan generates an initial daily plan based on goals.
func GenerateInitialPlan(goals []storage.Goal, attachments []storage.GoalAttachment, availableHours float64, startTime string) ([]storage.Task, error) {
	return activeProvider.GenerateInitialPlan(goals, attachments, availableHours, startTime)
}

// RegenerateAfterCheckin regenerates plan after evening check-in.
func RegenerateAfterCheckin(checkin storage.CheckIn, recentHistory []storage.DailyPlan, upcomingEvents []storage.Event) ([]storage.Task, error) {
	return activeProvider.RegenerateAfterCheckin(checkin, recentHistory, upcomingEvents)
}

// InsertEvent generates prep tasks when a new event is added.
func InsertEvent(event storage.Event, upcomingPlans []storage.DailyPlan) ([]storage.Task, error) {
	return activeProvider.InsertEvent(event, upcomingPlans)
}

// ParseGoalText parses free-form goal text into a structured ParsedGoal.
func ParseGoalText(rawInput string) (ParsedGoal, error) {
	return activeProvider.ParseGoalText(rawInput)
}

// defaultTasks returns a fallback plan when AI is unavailable.
func defaultTasks() []storage.Task {
	return []storage.Task{
		{Title: "Morning review", Description: "Review goals and plan for the day", Status: "pending", SortOrder: 1, SuggestedStart: "09:00", SuggestedEnd: "09:30"},
		{Title: "Deep work block", Description: "Focus on your primary goal", Status: "pending", SortOrder: 2, SuggestedStart: "09:30", SuggestedEnd: "11:30"},
		{Title: "Progress check", Description: "Review progress and adjust plan", Status: "pending", SortOrder: 3, SuggestedStart: "15:00", SuggestedEnd: "15:30"},
		{Title: "Evening wrap-up", Description: "Summarize accomplishments and plan tomorrow", Status: "pending", SortOrder: 4, SuggestedStart: "17:00", SuggestedEnd: "17:30"},
	}
}
