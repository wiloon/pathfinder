package ai

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"pathfinder-api/storage"
)

// OpenClawProvider routes AI work to a self-hosted OpenClaw backend via webhooks.
type OpenClawProvider struct {
	SyncURL       string
	WebhookURL    string
	WebhookSecret string
	ServiceToken  string
}

// webhookPayload is the envelope sent for all webhook events.
type webhookPayload struct {
	Event     string      `json:"event"`
	UserID    string      `json:"user_id"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// sendWebhook POSTs payload to WebhookURL with HMAC-SHA256 signature.
// Returns an error if the webhook URL is empty, delivery fails, or OpenClaw
// returns a non-2xx status. Caller must propagate the error to fail the
// user's operation (per delivery-confirmation semantics).
func (p *OpenClawProvider) sendWebhook(eventType string, data interface{}, userID string) error {
	if p.WebhookURL == "" {
		return nil // webhook not configured — skip silently
	}

	payload := webhookPayload{
		Event:     eventType,
		UserID:    userID,
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      data,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("sendWebhook %s: marshal: %w", eventType, err)
	}

	req, err := http.NewRequest(http.MethodPost, p.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("sendWebhook %s: new request: %w", eventType, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Pathfinder-Event", eventType)

	if p.WebhookSecret != "" {
		mac := hmac.New(sha256.New, []byte(p.WebhookSecret))
		mac.Write(body)
		req.Header.Set("X-Pathfinder-Signature", fmt.Sprintf("sha256=%x", mac.Sum(nil)))
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sendWebhook %s: do request: %w", eventType, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("sendWebhook %s: OpenClaw returned %d", eventType, resp.StatusCode)
	}
	return nil
}

// planGenerateData is the payload for the plan.generate_requested event.
type planGenerateData struct {
	Date  string         `json:"date"`
	Goals []storage.Goal `json:"goals"`
}

// GenerateInitialPlan fires a plan.generate_requested webhook and returns
// defaultTasks. OpenClaw processes the plan async and will push tasks back
// via the service token API (#10/#11).
func (p *OpenClawProvider) GenerateInitialPlan(goals []storage.Goal, attachments []storage.GoalAttachment, availableHours float64, startTime string) ([]storage.Task, error) {
	userID := resolveUserID(goals)
	data := planGenerateData{
		Date:  time.Now().Format("2006-01-02"),
		Goals: goals,
	}
	if err := p.sendWebhook("plan.generate_requested", data, userID); err != nil {
		log.Printf("OpenClaw GenerateInitialPlan webhook error: %v", err)
		return nil, fmt.Errorf("OpenClaw webhook failed: %w", err)
	}
	return defaultTasks(), nil
}

// checkinData is the payload for the checkin.submitted event.
type checkinData struct {
	Date          string      `json:"date"`
	Completed     string      `json:"completed"`
	Blocked       string      `json:"blocked"`
	TomorrowFocus string      `json:"tomorrow_focus"`
	TaskSummary   taskSummary `json:"task_summary"`
}

type taskSummary struct {
	Total   int `json:"total"`
	Done    int `json:"done"`
	Skipped int `json:"skipped"`
	Pending int `json:"pending"`
}

// RegenerateAfterCheckin fires a checkin.submitted webhook and returns
// defaultTasks. OpenClaw processes the replan async.
func (p *OpenClawProvider) RegenerateAfterCheckin(checkin storage.CheckIn, recentHistory []storage.DailyPlan, upcomingEvents []storage.Event) ([]storage.Task, error) {
	summary := taskSummary{}
	if len(recentHistory) > 0 {
		today := recentHistory[0]
		for _, t := range today.Tasks {
			summary.Total++
			switch t.Status {
			case "done":
				summary.Done++
			case "skipped":
				summary.Skipped++
			default:
				summary.Pending++
			}
		}
	}

	data := checkinData{
		Date:          checkin.Date,
		Completed:     checkin.Completed,
		Blocked:       checkin.Blocked,
		TomorrowFocus: checkin.TomorrowFocus,
		TaskSummary:   summary,
	}
	if err := p.sendWebhook("checkin.submitted", data, checkin.UserID); err != nil {
		log.Printf("OpenClaw RegenerateAfterCheckin webhook error: %v", err)
		return nil, fmt.Errorf("OpenClaw webhook failed: %w", err)
	}
	return defaultTasks(), nil
}

// InsertEvent is a no-op for OpenClaw: event context reaches OpenClaw via
// the goal/checkin webhook stream. No separate webhook for event creation.
func (p *OpenClawProvider) InsertEvent(event storage.Event, upcomingPlans []storage.DailyPlan) ([]storage.Task, error) {
	return nil, nil
}

// chatCompletion sends a synchronous request to the OpenClaw Gateway's
// /v1/chat/completions endpoint and returns the assistant message content.
func (p *OpenClawProvider) chatCompletion(messages []Message) (string, error) {
	if p.SyncURL == "" {
		return "", fmt.Errorf("openclaw sync_url not configured")
	}

	reqBody := chatRequest{Model: "openclaw", Messages: messages}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("chatCompletion: marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, p.SyncURL, bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("chatCompletion: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.ServiceToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.ServiceToken)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("chatCompletion: do request: %w", err)
	}
	defer resp.Body.Close()

	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return "", fmt.Errorf("chatCompletion: unmarshal response: %w", err)
	}
	if cr.Error != nil {
		return "", fmt.Errorf("OpenClaw API error: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from OpenClaw")
	}
	return cr.Choices[0].Message.Content, nil
}

// ParseGoalText calls OpenClaw to parse free-form goal text into structured data.
func (p *OpenClawProvider) ParseGoalText(rawInput string) (ParsedGoal, error) {
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

	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return ParsedGoal{}, fmt.Errorf("ParseGoalText: no JSON object in response")
	}

	var result ParsedGoal
	if err := json.Unmarshal([]byte(raw[start:end+1]), &result); err != nil {
		return ParsedGoal{}, fmt.Errorf("ParseGoalText: unmarshal: %w", err)
	}
	if result.Weight < 1 {
		result.Weight = 1
	}
	if result.Weight > 10 {
		result.Weight = 10
	}
	if result.Title == "" {
		result.Title = rawInput
		if len(result.Title) > 80 {
			result.Title = result.Title[:80]
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

// resolveUserID extracts the user ID from the first goal, or returns empty string.
func resolveUserID(goals []storage.Goal) string {
	if len(goals) > 0 {
		return goals[0].UserID
	}
	return ""
}
