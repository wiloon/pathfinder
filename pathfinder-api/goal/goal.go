package goal

import (
	"encoding/base64"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"pathfinder-api/ai"
	"pathfinder-api/storage"
)

// CreateGoal handles POST /api/goals
func CreateGoal(c *gin.Context) {
	userID := c.GetString("user_id")
	title := c.PostForm("title")
	description := c.PostForm("description")
	goalType := c.PostForm("type")
	status := c.PostForm("status")
	timeline := c.PostForm("timeline")

	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}
	if goalType == "" {
		goalType = "secondary"
	}
	if status == "" {
		status = "active"
	}

	g := storage.Goal{
		UserID:      userID,
		Title:       title,
		Description: description,
		Type:        goalType,
		Status:      status,
		Timeline:    timeline,
	}

	if err := storage.DB.Create(&g).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Handle file attachments.
	form, _ := c.MultipartForm()
	var attachments []storage.GoalAttachment
	if form != nil {
		for _, fh := range form.File["attachments"] {
			f, err := fh.Open()
			if err != nil {
				continue
			}
			data, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				continue
			}
			mimeType := fh.Header.Get("Content-Type")
			if mimeType == "" {
				mimeType = "application/octet-stream"
			}
			att := storage.GoalAttachment{
				GoalID:   g.ID,
				Filename: fh.Filename,
				MimeType: mimeType,
				Data:     data,
			}
			storage.DB.Create(&att)
			att.DataBase64 = base64.StdEncoding.EncodeToString(data)
			attachments = append(attachments, att)
		}
	}

	// If this is the first primary goal, generate an initial plan.
	if goalType == "primary" {
		today := time.Now().Format("2006-01-02")
		var existing storage.DailyPlan
		err := storage.DB.Where("user_id = ? AND date = ?", userID, today).First(&existing).Error
		if err != nil {
			// No plan today — generate one.
			var goals []storage.Goal
			storage.DB.Where("user_id = ? AND status = ?", userID, "active").Find(&goals)
			tasks, _ := ai.GenerateInitialPlan(goals, attachments, 8, "09:00")

			plan := storage.DailyPlan{UserID: userID, Date: today}
			storage.DB.Create(&plan)
			for i := range tasks {
				tasks[i].PlanID = plan.ID
				storage.DB.Create(&tasks[i])
			}
		}
	}

	g.Attachments = attachments
	c.JSON(http.StatusCreated, g)
}

// ListGoals handles GET /api/goals
func ListGoals(c *gin.Context) {
	userID := c.GetString("user_id")
	var goals []storage.Goal
	storage.DB.Where("user_id = ?", userID).Preload("Attachments").Find(&goals)

	// Populate base64 data for attachments.
	for i := range goals {
		for j := range goals[i].Attachments {
			goals[i].Attachments[j].DataBase64 = base64.StdEncoding.EncodeToString(goals[i].Attachments[j].Data)
		}
	}
	c.JSON(http.StatusOK, goals)
}

// UpdateGoal handles PUT /api/goals/:id
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
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Type        *string `json:"type"`
		Status      *string `json:"status"`
		Timeline    *string `json:"timeline"`
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
	if body.Type != nil {
		g.Type = *body.Type
	}
	if body.Status != nil {
		g.Status = *body.Status
	}
	if body.Timeline != nil {
		g.Timeline = *body.Timeline
	}

	storage.DB.Save(&g)
	c.JSON(http.StatusOK, g)
}

// DeleteGoal handles DELETE /api/goals/:id
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

// SetPrimaryGoal handles PUT /api/goals/:id/primary
func SetPrimaryGoal(c *gin.Context) {
	userID := c.GetString("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Set all goals to secondary first.
	storage.DB.Model(&storage.Goal{}).Where("user_id = ?", userID).Update("type", "secondary")

	// Set the target goal to primary.
	var g storage.Goal
	if err := storage.DB.Where("id = ? AND user_id = ?", id, userID).First(&g).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "goal not found"})
		return
	}
	g.Type = "primary"
	storage.DB.Save(&g)
	c.JSON(http.StatusOK, g)
}
