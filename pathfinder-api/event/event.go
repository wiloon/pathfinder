package event

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"pathfinder-api/ai"
	"pathfinder-api/storage"
)

// ListEvents handles GET /api/events
func ListEvents(c *gin.Context) {
	userID := c.GetString("user_id")
	var events []storage.Event
	storage.DB.Where("user_id = ? AND status = ?", userID, "upcoming").
		Preload("Attachments").
		Order("event_date asc").
		Find(&events)
	c.JSON(http.StatusOK, events)
}

// CreateEvent handles POST /api/events
func CreateEvent(c *gin.Context) {
	userID := c.GetString("user_id")
	title := c.PostForm("title")
	description := c.PostForm("description")
	eventDate := c.PostForm("event_date")

	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}
	if eventDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event_date is required"})
		return
	}

	ev := storage.Event{
		UserID:      userID,
		Title:       title,
		Description: description,
		EventDate:   eventDate,
		Status:      "upcoming",
	}
	if err := storage.DB.Create(&ev).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Handle file attachments.
	form, _ := c.MultipartForm()
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
			att := storage.EventAttachment{
				EventID:  ev.ID,
				Filename: fh.Filename,
				MimeType: mimeType,
				Data:     data,
			}
			storage.DB.Create(&att)
			ev.Attachments = append(ev.Attachments, att)
		}
	}

	// Generate prep tasks for tomorrow's plan.
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	var upcomingPlans []storage.DailyPlan
	storage.DB.Where("user_id = ? AND date >= ?", userID, tomorrow).
		Preload("Tasks").Limit(3).Find(&upcomingPlans)

	prepTasks, _ := ai.InsertEvent(ev, upcomingPlans)
	if len(prepTasks) > 0 {
		// Add prep tasks to tomorrow's plan.
		var tomorrowPlan storage.DailyPlan
		if err := storage.DB.Where("user_id = ? AND date = ?", userID, tomorrow).First(&tomorrowPlan).Error; err != nil {
			tomorrowPlan = storage.DailyPlan{UserID: userID, Date: tomorrow}
			storage.DB.Create(&tomorrowPlan)
		}

		// Find the current max sort_order.
		var maxOrder int
		storage.DB.Model(&storage.Task{}).Where("plan_id = ?", tomorrowPlan.ID).
			Select("COALESCE(MAX(sort_order), 0)").Scan(&maxOrder)

		for i := range prepTasks {
			prepTasks[i].PlanID = tomorrowPlan.ID
			prepTasks[i].SortOrder = maxOrder + i + 1
			storage.DB.Create(&prepTasks[i])
		}
	}

	c.JSON(http.StatusCreated, ev)
}

// DeleteEvent handles DELETE /api/events/:id
func DeleteEvent(c *gin.Context) {
	userID := c.GetString("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var ev storage.Event
	if err := storage.DB.Where("id = ? AND user_id = ?", id, userID).First(&ev).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	storage.DB.Where("event_id = ?", ev.ID).Delete(&storage.EventAttachment{})
	storage.DB.Delete(&ev)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// SubmitRetro handles POST /api/events/:id/retro
func SubmitRetro(c *gin.Context) {
	userID := c.GetString("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var ev storage.Event
	if err := storage.DB.Where("id = ? AND user_id = ?", id, userID).First(&ev).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	var body struct {
		RetroNote string `json:"retro_note"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ev.RetroNote = body.RetroNote
	ev.Status = "completed"
	storage.DB.Save(&ev)
	c.JSON(http.StatusOK, ev)
}
