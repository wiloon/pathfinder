package checkin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"pathfinder-api/ai"
	"pathfinder-api/storage"
)

const userID = "local"

// GetTodayCheckin handles GET /api/checkin/today
func GetTodayCheckin(c *gin.Context) {
	today := time.Now().Format("2006-01-02")

	var ci storage.CheckIn
	err := storage.DB.Where("user_id = ? AND date = ?", userID, today).First(&ci).Error
	if err != nil {
		// Return empty check-in structure.
		c.JSON(http.StatusOK, storage.CheckIn{UserID: userID, Date: today})
		return
	}
	c.JSON(http.StatusOK, ci)
}

// SubmitCheckin handles POST /api/checkin
func SubmitCheckin(c *gin.Context) {
	var body struct {
		Date          string `json:"date"`
		Completed     string `json:"completed"`
		Blocked       string `json:"blocked"`
		TomorrowFocus string `json:"tomorrow_focus"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	today := time.Now().Format("2006-01-02")
	if body.Date == "" {
		body.Date = today
	}

	// Upsert check-in.
	var ci storage.CheckIn
	storage.DB.Where("user_id = ? AND date = ?", userID, body.Date).First(&ci)
	ci.UserID = userID
	ci.Date = body.Date
	ci.Completed = body.Completed
	ci.Blocked = body.Blocked
	ci.TomorrowFocus = body.TomorrowFocus

	if ci.ID == 0 {
		storage.DB.Create(&ci)
	} else {
		storage.DB.Save(&ci)
	}

	// Regenerate tomorrow's plan.
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	// Fetch recent daily plan history (last 7 days).
	var recentHistory []storage.DailyPlan
	storage.DB.Where("user_id = ?", userID).
		Order("date desc").Limit(7).
		Preload("Tasks").Find(&recentHistory)

	// Fetch upcoming events.
	var upcomingEvents []storage.Event
	storage.DB.Where("user_id = ? AND status = ? AND event_date >= ?", userID, "upcoming", tomorrow).
		Find(&upcomingEvents)

	tasks, _ := ai.RegenerateAfterCheckin(ci, recentHistory, upcomingEvents)

	// Upsert tomorrow's plan.
	var tomorrowPlan storage.DailyPlan
	if err := storage.DB.Where("user_id = ? AND date = ?", userID, tomorrow).First(&tomorrowPlan).Error; err != nil {
		tomorrowPlan = storage.DailyPlan{UserID: userID, Date: tomorrow}
		storage.DB.Create(&tomorrowPlan)
	} else {
		storage.DB.Where("plan_id = ?", tomorrowPlan.ID).Delete(&storage.Task{})
	}

	for i := range tasks {
		tasks[i].PlanID = tomorrowPlan.ID
		storage.DB.Create(&tasks[i])
	}
	tomorrowPlan.Tasks = tasks

	c.JSON(http.StatusOK, gin.H{
		"checkin":       ci,
		"tomorrow_plan": tomorrowPlan,
	})
}
