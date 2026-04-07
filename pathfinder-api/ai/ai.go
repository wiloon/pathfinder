package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"pathfinder-api/storage"
)

// Config holds MiniMax API configuration.
type Config struct {
	APIKey  string
	Model   string
	BaseURL string
}

var Cfg Config

func Init(cfg Config) {
	Cfg = cfg
}

// Message represents a chat message.
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []ContentPart
}

// ContentPart is one piece of a multimodal message.
type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL holds a base64-encoded image URL.
type ImageURL struct {
	URL string `json:"url"`
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ChatCompletion sends messages to MiniMax and returns the response text.
func ChatCompletion(messages []Message) (string, error) {
	if Cfg.APIKey == "" {
		return "", fmt.Errorf("AI API key not configured")
	}

	reqBody := chatRequest{
		Model:    Cfg.Model,
		Messages: messages,
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := strings.TrimRight(Cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+Cfg.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var cr chatResponse
	if err := json.Unmarshal(data, &cr); err != nil {
		return "", err
	}
	if cr.Error != nil {
		return "", fmt.Errorf("MiniMax API error: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from API")
	}
	return cr.Choices[0].Message.Content, nil
}

// parseTasks extracts a JSON tasks array from the AI response.
func parseTasks(raw string) []storage.Task {
	// Try to find a JSON array in the response.
	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start == -1 || end == -1 || end <= start {
		return nil
	}
	jsonStr := raw[start : end+1]

	var tasks []storage.Task
	if err := json.Unmarshal([]byte(jsonStr), &tasks); err != nil {
		log.Printf("failed to parse AI tasks: %v\nraw: %s", err, jsonStr)
		return nil
	}
	return tasks
}

// GenerateInitialPlan generates an initial daily plan based on goals.
func GenerateInitialPlan(goals []storage.Goal, attachments []storage.GoalAttachment, availableHours float64, startTime string) ([]storage.Task, error) {
	if len(goals) == 0 {
		return defaultTasks(), nil
	}

	goalsJSON, _ := json.MarshalIndent(goals, "", "  ")

	prompt := fmt.Sprintf(`You are a personal productivity coach. Generate a realistic daily task plan.

Goals:
%s

Available hours today: %.1f
Preferred start time: %s

Return ONLY a JSON array of tasks with this structure (no extra text):
[
  {
    "title": "Task title",
    "description": "Short description",
    "status": "pending",
    "sort_order": 1,
    "suggested_start": "HH:MM",
    "suggested_end": "HH:MM"
  }
]

Generate 4-8 focused tasks that make meaningful progress on the goals.`, string(goalsJSON), availableHours, startTime)

	messages := []Message{
		{Role: "user", Content: prompt},
	}

	// Include image attachments if any.
	if len(attachments) > 0 {
		var parts []ContentPart
		parts = append(parts, ContentPart{Type: "text", Text: prompt})
		for _, att := range attachments {
			if strings.HasPrefix(att.MimeType, "image/") && len(att.DataBase64) > 0 {
				parts = append(parts, ContentPart{
					Type: "image_url",
					ImageURL: &ImageURL{
						URL: fmt.Sprintf("data:%s;base64,%s", att.MimeType, att.DataBase64),
					},
				})
			}
		}
		messages[0].Content = parts
	}

	raw, err := ChatCompletion(messages)
	if err != nil {
		log.Printf("AI GenerateInitialPlan error: %v", err)
		return defaultTasks(), nil
	}

	tasks := parseTasks(raw)
	if len(tasks) == 0 {
		return defaultTasks(), nil
	}
	for i := range tasks {
		tasks[i].Status = "pending"
		if tasks[i].SortOrder == 0 {
			tasks[i].SortOrder = i + 1
		}
	}
	return tasks, nil
}

// RegenerateAfterCheckin regenerates plan after evening check-in.
func RegenerateAfterCheckin(checkin storage.CheckIn, recentHistory []storage.DailyPlan, upcomingEvents []storage.Event) ([]storage.Task, error) {
	historyJSON, _ := json.MarshalIndent(recentHistory, "", "  ")
	eventsJSON, _ := json.MarshalIndent(upcomingEvents, "", "  ")

	prompt := fmt.Sprintf(`You are a personal productivity coach. Based on today's check-in and upcoming events, generate tomorrow's task plan.

Today's Check-in:
- Completed: %s
- Blocked: %s
- Tomorrow's Focus: %s

Recent history:
%s

Upcoming events:
%s

Return ONLY a JSON array of tasks:
[
  {
    "title": "Task title",
    "description": "Short description",
    "status": "pending",
    "sort_order": 1,
    "suggested_start": "HH:MM",
    "suggested_end": "HH:MM"
  }
]

Generate 4-8 focused tasks for tomorrow.`,
		checkin.Completed, checkin.Blocked, checkin.TomorrowFocus,
		string(historyJSON), string(eventsJSON))

	messages := []Message{
		{Role: "user", Content: prompt},
	}

	raw, err := ChatCompletion(messages)
	if err != nil {
		log.Printf("AI RegenerateAfterCheckin error: %v", err)
		return defaultTasks(), nil
	}

	tasks := parseTasks(raw)
	if len(tasks) == 0 {
		return defaultTasks(), nil
	}
	for i := range tasks {
		tasks[i].Status = "pending"
		if tasks[i].SortOrder == 0 {
			tasks[i].SortOrder = i + 1
		}
	}
	return tasks, nil
}

// InsertEvent generates prep tasks when a new event is added.
func InsertEvent(event storage.Event, upcomingPlans []storage.DailyPlan) ([]storage.Task, error) {
	plansJSON, _ := json.MarshalIndent(upcomingPlans, "", "  ")

	prompt := fmt.Sprintf(`You are a personal productivity coach. A new event has been added. Generate preparation tasks for tomorrow's plan.

Event:
- Title: %s
- Description: %s
- Date: %s

Existing upcoming plans:
%s

Return ONLY a JSON array of preparation tasks (2-4 tasks):
[
  {
    "title": "Task title",
    "description": "Short description",
    "status": "pending",
    "sort_order": 1,
    "suggested_start": "HH:MM",
    "suggested_end": "HH:MM"
  }
]`,
		event.Title, event.Description, event.EventDate, string(plansJSON))

	messages := []Message{
		{Role: "user", Content: prompt},
	}

	raw, err := ChatCompletion(messages)
	if err != nil {
		log.Printf("AI InsertEvent error: %v", err)
		return nil, nil
	}

	tasks := parseTasks(raw)
	for i := range tasks {
		tasks[i].Status = "pending"
		if tasks[i].SortOrder == 0 {
			tasks[i].SortOrder = i + 1
		}
	}
	return tasks, nil
}

func defaultTasks() []storage.Task {
	return []storage.Task{
		{Title: "Morning review", Description: "Review goals and plan for the day", Status: "pending", SortOrder: 1, SuggestedStart: "09:00", SuggestedEnd: "09:30"},
		{Title: "Deep work block", Description: "Focus on your primary goal", Status: "pending", SortOrder: 2, SuggestedStart: "09:30", SuggestedEnd: "11:30"},
		{Title: "Progress check", Description: "Review progress and adjust plan", Status: "pending", SortOrder: 3, SuggestedStart: "15:00", SuggestedEnd: "15:30"},
		{Title: "Evening wrap-up", Description: "Summarize accomplishments and plan tomorrow", Status: "pending", SortOrder: 4, SuggestedStart: "17:00", SuggestedEnd: "17:30"},
	}
}
