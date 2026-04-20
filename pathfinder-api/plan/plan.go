package plan

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"pathfinder-api/ai"
	"pathfinder-api/storage"
)

// GetTodayPlan handles GET /api/plan/today
func GetTodayPlan(c *gin.Context) {
	userID := c.GetString("user_id")
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

		tasks, _ := ai.GenerateInitialPlan(goals, nil, availableHours(userID), "09:00")
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
	userID := c.GetString("user_id")
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

	tasks, _ := ai.GenerateInitialPlan(goals, nil, availableHours(userID), "09:00")
	for i := range tasks {
		tasks[i].PlanID = plan.ID
		storage.DB.Create(&tasks[i])
	}

	storage.DB.Where("plan_id = ?", plan.ID).Order("sort_order asc").Find(&plan.Tasks)
	c.JSON(http.StatusOK, plan)
}

// DayAgenda aggregates goals, events, and tasks for a single date.
type DayAgenda struct {
	Date   string          `json:"date"`
	Goals  []storage.Goal  `json:"goals"`
	Events []storage.Event `json:"events"`
	Tasks  []storage.Task  `json:"tasks"`
}

// WeekAgendaResponse is the payload for GET /api/agenda/week.
type WeekAgendaResponse struct {
	Days        []DayAgenda    `json:"days"`
	Unscheduled []storage.Goal `json:"unscheduled"`
}

var datePattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)

// GetWeekAgenda handles GET /api/agenda/week?start=YYYY-MM-DD
// Returns 7 days of aggregated goals, events, and tasks.
// Goals with a YYYY-MM-DD timeline matching a day in the window are placed on
// that day; all other active goals go into the Unscheduled bucket.
func GetWeekAgenda(c *gin.Context) {
	userID := c.GetString("user_id")

	startStr := c.DefaultQuery("start", time.Now().Format("2006-01-02"))
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		start = time.Now().Truncate(24 * time.Hour)
	}

	dates := make([]string, 7)
	for i := range dates {
		dates[i] = start.AddDate(0, 0, i).Format("2006-01-02")
	}
	endDate := dates[6]

	// Fetch all active goals.
	var goals []storage.Goal
	storage.DB.Where("user_id = ? AND status = ?", userID, "active").Find(&goals)

	// Fetch events in the window.
	var events []storage.Event
	storage.DB.Where("user_id = ? AND event_date >= ? AND event_date <= ?", userID, dates[0], endDate).Find(&events)

	// Fetch daily plans in the window, then load tasks per plan.
	var plans []storage.DailyPlan
	storage.DB.Where("user_id = ? AND date >= ? AND date <= ?", userID, dates[0], endDate).Find(&plans)
	for i := range plans {
		storage.DB.Where("plan_id = ?", plans[i].ID).Order("sort_order asc").Find(&plans[i].Tasks)
	}

	// Build per-day buckets.
	type dayBucket struct {
		Goals  []storage.Goal
		Events []storage.Event
		Tasks  []storage.Task
	}
	buckets := make(map[string]*dayBucket, 7)
	for _, d := range dates {
		buckets[d] = &dayBucket{
			Goals:  []storage.Goal{},
			Events: []storage.Event{},
			Tasks:  []storage.Task{},
		}
	}

	// Place events by event_date.
	for _, e := range events {
		if b, ok := buckets[e.EventDate]; ok {
			b.Events = append(b.Events, e)
		}
	}

	// Place tasks from daily plans.
	for _, p := range plans {
		if b, ok := buckets[p.Date]; ok {
			b.Tasks = append(b.Tasks, p.Tasks...)
		}
	}

	// Place goals by timeline date; unmatched go to Unscheduled.
	var unscheduled []storage.Goal
	for _, g := range goals {
		match := datePattern.FindString(g.Timeline)
		if match != "" {
			if b, ok := buckets[match]; ok {
				b.Goals = append(b.Goals, g)
				continue
			}
		}
		unscheduled = append(unscheduled, g)
	}
	if unscheduled == nil {
		unscheduled = []storage.Goal{}
	}

	// Assemble ordered response.
	result := WeekAgendaResponse{Unscheduled: unscheduled}
	for _, d := range dates {
		b := buckets[d]
		result.Days = append(result.Days, DayAgenda{
			Date:   d,
			Goals:  b.Goals,
			Events: b.Events,
			Tasks:  b.Tasks,
		})
	}

	c.JSON(http.StatusOK, result)
}

// availableHours returns the user's configured daily available hours,
// falling back to 8.0 if no profile exists.
func availableHours(userID string) float64 {
	var profile storage.UserProfile
	if err := storage.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return 8.0
	}
	if profile.DailyAvailableHours <= 0 {
		return 8.0
	}
	return profile.DailyAvailableHours
}

type taskPatchBody struct {
	Status         *string `json:"status"`
	SortOrder      *int    `json:"sort_order"`
	Title          *string `json:"title"`
	Description    *string `json:"description"`
	SuggestedStart *string `json:"suggested_start"`
	SuggestedEnd   *string `json:"suggested_end"`
}

func applyTaskPatch(task *storage.Task, body taskPatchBody) {
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

	var body taskPatchBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	applyTaskPatch(&task, body)

	storage.DB.Save(&task)
	c.JSON(http.StatusOK, task)
}

// DeleteTask handles DELETE /api/tasks/:id
func DeleteTask(c *gin.Context) {
	userID := c.GetString("user_id")
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

	// Verify ownership via the plan.
	var plan storage.DailyPlan
	if err := storage.DB.First(&plan, task.PlanID).Error; err != nil || plan.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	storage.DB.Delete(&task)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// PatchTask handles PATCH /api/agent/tasks/:id
// Same semantics as UpdateTask but accessible via service token auth.
// Ownership check: task must belong to a plan owned by the requesting user.
func PatchTask(c *gin.Context) {
	userID := c.GetString("user_id")
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

	// Verify the task belongs to this user via its plan.
	var plan storage.DailyPlan
	if err := storage.DB.First(&plan, task.PlanID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	if plan.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var body taskPatchBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	applyTaskPatch(&task, body)

	if err := storage.DB.Save(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update task"})
		return
	}
	c.JSON(http.StatusOK, task)
}

// createTaskForDate finds or creates the DailyPlan for the given date and userID,
// then inserts a new Task with the provided fields.
// sort_order is set to max(existing) + 1.
func createTaskForDate(userID, date, title, description string, goalID *uint, suggestedStart, suggestedEnd string) (storage.Task, error) {
	var plan storage.DailyPlan
	err := storage.DB.Where("user_id = ? AND date = ?", userID, date).First(&plan).Error
	if err != nil {
		plan = storage.DailyPlan{UserID: userID, Date: date}
		if err := storage.DB.Create(&plan).Error; err != nil {
			return storage.Task{}, fmt.Errorf("createTaskForDate: create plan: %w", err)
		}
	}

	var maxOrder int
	storage.DB.Model(&storage.Task{}).
		Where("plan_id = ?", plan.ID).
		Select("COALESCE(MAX(sort_order), 0)").
		Scan(&maxOrder)

	task := storage.Task{
		PlanID:         plan.ID,
		GoalID:         goalID,
		Title:          title,
		Description:    description,
		Status:         "pending",
		SortOrder:      maxOrder + 1,
		SuggestedStart: suggestedStart,
		SuggestedEnd:   suggestedEnd,
	}
	if err := storage.DB.Create(&task).Error; err != nil {
		return storage.Task{}, fmt.Errorf("createTaskForDate: create task: %w", err)
	}
	return task, nil
}

// CreateTask handles POST /api/agent/tasks
// Accepts: title (required), description, goal_id, date (YYYY-MM-DD, defaults to today),
// suggested_start, suggested_end (HH:MM).
func CreateTask(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		Title          string `json:"title"`
		Description    string `json:"description"`
		GoalID         *uint  `json:"goal_id"`
		Date           string `json:"date"`
		SuggestedStart string `json:"suggested_start"`
		SuggestedEnd   string `json:"suggested_end"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}
	if len(body.Title) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title too long (max 200)"})
		return
	}

	date := body.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date must be YYYY-MM-DD"})
		return
	}

	task, err := createTaskForDate(userID, date, body.Title, body.Description, body.GoalID, body.SuggestedStart, body.SuggestedEnd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
		return
	}
	c.JSON(http.StatusCreated, task)
}

// CreateTaskQuick handles POST /api/plan/tasks/quick
// User-auth endpoint for creating a single task on a specific date (or today).
// Accepts: title (required), description, date (YYYY-MM-DD, defaults to today), goal_id.
func CreateTaskQuick(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Date        string `json:"date"`
		GoalID      *uint  `json:"goal_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}
	if len(body.Title) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title too long (max 200)"})
		return
	}

	date := body.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date must be YYYY-MM-DD"})
		return
	}

	task, err := createTaskForDate(userID, date, body.Title, body.Description, body.GoalID, "", "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
		return
	}
	c.JSON(http.StatusCreated, task)
}
