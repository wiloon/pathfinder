package goal

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"pathfinder-api/ai"
	"pathfinder-api/storage"
)

// encodeTags marshals a string slice to a compact JSON array string.
// Returns "[]" for nil or empty input.
func encodeTags(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	b, err := json.Marshal(tags)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// CreateGoal handles POST /api/goals
// Accepts JSON: title (required), description, weight (1–10, default 5),
// tags ([]string), status, timeline.
func CreateGoal(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Weight      *int     `json:"weight"`
		Tags        []string `json:"tags"`
		Status      string   `json:"status"`
		Timeline    string   `json:"timeline"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if body.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}

	weight := 5
	if body.Weight != nil {
		weight = *body.Weight
	}
	if weight < 1 || weight > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "weight must be between 1 and 10"})
		return
	}

	status := body.Status
	if status == "" {
		status = "active"
	}

	g := storage.Goal{
		UserID:      userID,
		Title:       body.Title,
		Description: body.Description,
		Weight:      weight,
		Tags:        encodeTags(body.Tags),
		Status:      status,
		Timeline:    body.Timeline,
	}

	if err := storage.DB.Create(&g).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Errorf("createGoal: %w", err).Error()})
		return
	}

	// If no plan exists today, generate one from all active goals.
	today := time.Now().Format("2006-01-02")
	if err := storage.DB.Where("user_id = ? AND date = ?", userID, today).First(&storage.DailyPlan{}).Error; err != nil {
		var goals []storage.Goal
		storage.DB.Where("user_id = ? AND status = ?", userID, "active").Find(&goals)

		var profile storage.UserProfile
		hours := 8.0
		if storage.DB.Where("user_id = ?", userID).First(&profile).Error == nil && profile.DailyAvailableHours > 0 {
			hours = profile.DailyAvailableHours
		}

		tasks, _ := ai.GenerateInitialPlan(goals, nil, hours, "09:00")

		plan := storage.DailyPlan{UserID: userID, Date: today}
		storage.DB.Create(&plan)
		for i := range tasks {
			tasks[i].PlanID = plan.ID
			storage.DB.Create(&tasks[i])
		}
	}

	c.JSON(http.StatusCreated, g)
}

// ListGoals handles GET /api/goals
func ListGoals(c *gin.Context) {
	userID := c.GetString("user_id")
	var goals []storage.Goal
	storage.DB.Where("user_id = ?", userID).Preload("Attachments").Find(&goals)

	for i := range goals {
		for j := range goals[i].Attachments {
			goals[i].Attachments[j].DataBase64 = base64.StdEncoding.EncodeToString(goals[i].Attachments[j].Data)
		}
	}
	c.JSON(http.StatusOK, goals)
}

// UpdateGoal handles PUT /api/goals/:id
// Accepts JSON; all fields optional (patch semantics).
func UpdateGoal(c *gin.Context) {
	userID := c.GetString("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var g storage.Goal
	if err := storage.DB.Where("id = ? AND user_id = ?", id, userID).First(&g).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "goal not found"})
		return
	}

	var body struct {
		Title       *string  `json:"title"`
		Description *string  `json:"description"`
		Weight      *int     `json:"weight"`
		Tags        []string `json:"tags"`
		Status      *string  `json:"status"`
		Timeline    *string  `json:"timeline"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if body.Title != nil {
		g.Title = *body.Title
	}
	if body.Description != nil {
		g.Description = *body.Description
	}
	if body.Weight != nil {
		if *body.Weight < 1 || *body.Weight > 10 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "weight must be between 1 and 10"})
			return
		}
		g.Weight = *body.Weight
	}
	if body.Tags != nil {
		g.Tags = encodeTags(body.Tags)
	}
	if body.Status != nil {
		g.Status = *body.Status
	}
	if body.Timeline != nil {
		g.Timeline = *body.Timeline
	}

	if err := storage.DB.Save(&g).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Errorf("updateGoal: %w", err).Error()})
		return
	}
	c.JSON(http.StatusOK, g)
}

// PatchGoal handles PATCH /api/agent/goals/:id
// Service-token-auth only. Currently supports weight adjustment (1–10).
func PatchGoal(c *gin.Context) {
	userID := c.GetString("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var g storage.Goal
	if err := storage.DB.Where("id = ? AND user_id = ?", id, userID).First(&g).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "goal not found"})
		return
	}

	var body struct {
		Weight *int `json:"weight"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Weight == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "weight is required"})
		return
	}
	if *body.Weight < 1 || *body.Weight > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "weight must be between 1 and 10"})
		return
	}

	g.Weight = *body.Weight
	if err := storage.DB.Save(&g).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Errorf("patchGoal: %w", err).Error()})
		return
	}
	c.JSON(http.StatusOK, g)
}
// ParseGoal handles POST /api/goals/parse
// Accepts JSON: raw_input (required, max 500 chars).
// Calls AI to parse free-form goal text; does NOT save to DB.
func ParseGoal(c *gin.Context) {
	var body struct {
		RawInput string `json:"raw_input"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.RawInput == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "raw_input is required"})
		return
	}
	if len(body.RawInput) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "raw_input must be 500 characters or fewer"})
		return
	}

	parsed, err := ai.ParseGoalText(body.RawInput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Errorf("parseGoal: %w", err).Error()})
		return
	}
	c.JSON(http.StatusOK, parsed)
}

func DeleteGoal(c *gin.Context) {
	userID := c.GetString("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var g storage.Goal
	if err := storage.DB.Where("id = ? AND user_id = ?", id, userID).First(&g).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "goal not found"})
		return
	}

	storage.DB.Where("goal_id = ?", g.ID).Delete(&storage.GoalAttachment{})
	storage.DB.Delete(&g)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
