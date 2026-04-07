package plan

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"pathfinder-api/ai"
	"pathfinder-api/storage"
)

const userID = "local"

// GetTodayPlan handles GET /api/plan/today
func GetTodayPlan(c *gin.Context) {
	today := time.Now().Format("2006-01-02")

	var plan storage.DailyPlan
	err := storage.DB.Where("user_id = ? AND date = ?", userID, today).
		Preload("Tasks").First(&plan).Error

	if err != nil {
		// No plan — generate one.
		plan = storage.DailyPlan{UserID: userID, Date: today}
		storage.DB.Create(&plan)

		var goals []storage.Goal
		storage.DB.Where("user_id = ? AND status = ?", userID, "active").Find(&goals)

		tasks, _ := ai.GenerateInitialPlan(goals, nil, 8, "09:00")
		for i := range tasks {
			tasks[i].PlanID = plan.ID
			storage.DB.Create(&tasks[i])
		}
		plan.Tasks = tasks
	}

	// Sort tasks by sort_order.
	storage.DB.Where("plan_id = ?", plan.ID).Order("sort_order asc").Find(&plan.Tasks)
	c.JSON(http.StatusOK, plan)
}

// GeneratePlan handles POST /api/plan/generate
func GeneratePlan(c *gin.Context) {
	today := time.Now().Format("2006-01-02")

	var plan storage.DailyPlan
	if err := storage.DB.Where("user_id = ? AND date = ?", userID, today).First(&plan).Error; err != nil {
		plan = storage.DailyPlan{UserID: userID, Date: today}
		storage.DB.Create(&plan)
	} else {
		// Delete existing tasks.
		storage.DB.Where("plan_id = ?", plan.ID).Delete(&storage.Task{})
	}

	var goals []storage.Goal
	storage.DB.Where("user_id = ? AND status = ?", userID, "active").Find(&goals)

	tasks, _ := ai.GenerateInitialPlan(goals, nil, 8, "09:00")
	for i := range tasks {
		tasks[i].PlanID = plan.ID
		storage.DB.Create(&tasks[i])
	}

	storage.DB.Where("plan_id = ?", plan.ID).Order("sort_order asc").Find(&plan.Tasks)
	c.JSON(http.StatusOK, plan)
}

// UpdateTask handles PUT /api/tasks/:id
func UpdateTask(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var task storage.Task
	if err := storage.DB.First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	var body struct {
		Status         *string `json:"status"`
		SortOrder      *int    `json:"sort_order"`
		Title          *string `json:"title"`
		Description    *string `json:"description"`
		SuggestedStart *string `json:"suggested_start"`
		SuggestedEnd   *string `json:"suggested_end"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if body.Status != nil {
		task.Status = *body.Status
	}
	if body.SortOrder != nil {
		task.SortOrder = *body.SortOrder
	}
	if body.Title != nil {
		task.Title = *body.Title
	}
	if body.Description != nil {
		task.Description = *body.Description
	}
	if body.SuggestedStart != nil {
		task.SuggestedStart = *body.SuggestedStart
	}
	if body.SuggestedEnd != nil {
		task.SuggestedEnd = *body.SuggestedEnd
	}

	storage.DB.Save(&task)
	c.JSON(http.StatusOK, task)
}
