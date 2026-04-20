package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"pathfinder-api/storage"
)

// MiniMaxProvider calls the MiniMax Chat API (OpenAI-compatible).
type MiniMaxProvider struct {
	APIKey  string
	Model   string
	BaseURL string
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

func (p *MiniMaxProvider) chatCompletion(messages []Message) (string, error) {
	if p.APIKey == "" {
		return "", fmt.Errorf("MiniMax API key not configured")
	}

	reqBody := chatRequest{Model: p.Model, Messages: messages}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("chatCompletion: marshal: %w", err)
	}

	url := strings.TrimRight(p.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("chatCompletion: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("chatCompletion: do request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("chatCompletion: read body: %w", err)
	}

	var cr chatResponse
	if err := json.Unmarshal(data, &cr); err != nil {
		return "", fmt.Errorf("chatCompletion: unmarshal response: %w", err)
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

func normalizeTasks(tasks []storage.Task) []storage.Task {
	for i := range tasks {
		tasks[i].Status = "pending"
		if tasks[i].SortOrder == 0 {
			tasks[i].SortOrder = i + 1
		}
	}
	return tasks
}

func (p *MiniMaxProvider) GenerateInitialPlan(goals []storage.Goal, attachments []storage.GoalAttachment, availableHours float64, startTime string) ([]storage.Task, error) {
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

	messages := []Message{{Role: "user", Content: prompt}}

	if len(attachments) > 0 {
		var parts []ContentPart
		parts = append(parts, ContentPart{Type: "text", Text: prompt})
		for _, att := range attachments {
			if strings.HasPrefix(att.MimeType, "image/") && len(att.DataBase64) > 0 {
				parts = append(parts, ContentPart{
					Type:     "image_url",
					ImageURL: &ImageURL{URL: fmt.Sprintf("data:%s;base64,%s", att.MimeType, att.DataBase64)},
				})
			}
		}
		messages[0].Content = parts
	}

	raw, err := p.chatCompletion(messages)
	if err != nil {
		log.Printf("MiniMax GenerateInitialPlan error: %v", err)
		return defaultTasks(), nil
	}

	tasks := parseTasks(raw)
	if len(tasks) == 0 {
		return defaultTasks(), nil
	}
	return normalizeTasks(tasks), nil
}

func (p *MiniMaxProvider) RegenerateAfterCheckin(checkin storage.CheckIn, recentHistory []storage.DailyPlan, upcomingEvents []storage.Event) ([]storage.Task, error) {
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

	raw, err := p.chatCompletion([]Message{{Role: "user", Content: prompt}})
	if err != nil {
		log.Printf("MiniMax RegenerateAfterCheckin error: %v", err)
		return defaultTasks(), nil
	}

	tasks := parseTasks(raw)
	if len(tasks) == 0 {
		return defaultTasks(), nil
	}
	return normalizeTasks(tasks), nil
}

func (p *MiniMaxProvider) InsertEvent(event storage.Event, upcomingPlans []storage.DailyPlan) ([]storage.Task, error) {
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

	raw, err := p.chatCompletion([]Message{{Role: "user", Content: prompt}})
	if err != nil {
		log.Printf("MiniMax InsertEvent error: %v", err)
		return nil, nil
	}

	tasks := parseTasks(raw)
	return normalizeTasks(tasks), nil
}

// ParseGoalText parses free-form goal text into a structured ParsedGoal using MiniMax.
func (p *MiniMaxProvider) ParseGoalText(rawInput string) (ParsedGoal, error) {
	today := time.Now().Format("2006-01-02")
	inputJSON, _ := json.Marshal(rawInput)
	prompt := fmt.Sprintf(`你是目标与任务拆解助手。今天是 %s。

用户描述了一件事情或要达成的目标。请分析并提取以下三类信息：

【1. 持久目标】这件事的核心目标是什么（长期持续追踪的目标）。
【2. 近期事件】有明确日期或时间点的具体行动（约定、截止日、去某处办事、提交材料等）。
【3. 今日准备】为了完成上述事件或目标，今天就必须开始做的准备工作。
  - 判断标准：如果等到事件当天才开始准备会来不及，那就是"今日准备"。
  - 例如：需要提前整理、打印、复印的材料；需要今天联系的人；需要今天完成的前置工作。
  - "今日准备"的 date 设为今天（%s）。

用户输入：%s

只返回如下 JSON 对象（不要任何额外文字或 markdown）：
{
  "title": "简洁的目标标题（最多80字）",
  "description": "对持久目标的完整描述",
  "weight": 5,
  "tags": ["tag1", "tag2"],
  "timeline": "目标的时间跨度，如 ongoing、2026年Q3，长期目标留空",
  "extracted_events": [
    {
      "title": "事件标题",
      "description": "需要做什么的详细说明",
      "event_date": "YYYY-MM-DD",
      "note": "用户原文中的时间表述，如 '明天上午'"
    }
  ],
  "extracted_tasks": [
    {
      "title": "准备任务标题",
      "description": "具体需要做什么",
      "date": "YYYY-MM-DD"
    }
  ]
}

规则：
- title 不能为空
- weight 为 1-10 的整数（5=中等优先级）
- tags 为 0-5 个小写关键词
- timeline 描述目标跨度，不是具体任务日期
- extracted_events：仅当描述中有明确日期的行动时填写；相对日期（明天、后天）从今天（%s）计算，结果用 YYYY-MM-DD 格式；日期不明确时 event_date 留空 ""
- extracted_tasks：今天就必须启动的准备工作，date 设为 %s；如无此类工作则为空数组 []
- 两个数组都可以为空 []`, today, today, string(inputJSON), today, today)

	raw, err := p.chatCompletion([]Message{{Role: "user", Content: prompt}})
	if err != nil {
		return ParsedGoal{}, fmt.Errorf("ParseGoalText: %w", err)
	}

	// Extract JSON object from response.
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return ParsedGoal{}, fmt.Errorf("ParseGoalText: no JSON object in response")
	}

	var result ParsedGoal
	if err := json.Unmarshal([]byte(raw[start:end+1]), &result); err != nil {
		return ParsedGoal{}, fmt.Errorf("ParseGoalText: unmarshal: %w", err)
	}

	// Clamp weight to 1–10.
	if result.Weight < 1 {
		result.Weight = 1
	}
	if result.Weight > 10 {
		result.Weight = 10
	}
	// Ensure title is non-empty.
	if result.Title == "" {
		if len(rawInput) > 80 {
			result.Title = rawInput[:80]
		} else {
			result.Title = rawInput
		}
	}
	if result.Tags == nil {
		result.Tags = []string{}
	}
	if result.ExtractedEvents == nil {
		result.ExtractedEvents = []ParsedEvent{}
	}
	if result.ExtractedTasks == nil {
		result.ExtractedTasks = []ParsedTask{}
	}
	return result, nil
}
