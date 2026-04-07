package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	aiPkg "pathfinder-api/ai"
	"pathfinder-api/checkin"
	"pathfinder-api/event"
	"pathfinder-api/goal"
	"pathfinder-api/middleware"
	"pathfinder-api/plan"
	"pathfinder-api/storage"
)

type Config struct {
	Server   ServerConfig   `toml:"server"`
	Database DatabaseConfig `toml:"database"`
	AI       AIConfig       `toml:"ai"`
}

type ServerConfig struct {
	Port          string `toml:"port"`
	SessionSecret string `toml:"session_secret"`
}

type DatabaseConfig struct {
	DSN string `toml:"dsn"`
}

type AIConfig struct {
	APIKey  string `toml:"api_key"`
	Model   string `toml:"model"`
	BaseURL string `toml:"base_url"`
}

func loadConfig(path string) Config {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	return cfg
}

func main() {
	cfgPath := "config.toml"
	if v := os.Getenv("CONFIG_PATH"); v != "" {
		cfgPath = v
	}
	cfg := loadConfig(cfgPath)

	storage.Init(cfg.Database.DSN)
	middleware.InitSession(cfg.Server.SessionSecret)
	aiPkg.Init(aiPkg.Config{
		APIKey:  cfg.AI.APIKey,
		Model:   cfg.AI.Model,
		BaseURL: cfg.AI.BaseURL,
	})

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))
	r.Use(middleware.Session())

	api := r.Group("/api")
	{
		// Goals
		api.POST("/goals", goal.CreateGoal)
		api.GET("/goals", goal.ListGoals)
		api.PUT("/goals/:id", goal.UpdateGoal)
		api.DELETE("/goals/:id", goal.DeleteGoal)
		api.PUT("/goals/:id/primary", goal.SetPrimaryGoal)

		// Plan
		api.GET("/plan/today", plan.GetTodayPlan)
		api.POST("/plan/generate", plan.GeneratePlan)
		api.PUT("/tasks/:id", plan.UpdateTask)

		// Events
		api.GET("/events", event.ListEvents)
		api.POST("/events", event.CreateEvent)
		api.DELETE("/events/:id", event.DeleteEvent)
		api.POST("/events/:id/retro", event.SubmitRetro)

		// Check-in
		api.GET("/checkin/today", checkin.GetTodayCheckin)
		api.POST("/checkin", checkin.SubmitCheckin)

		// Export / Import
		api.GET("/export", exportHandler)
		api.POST("/import", importHandler)
	}

	addr := ":" + cfg.Server.Port
	log.Printf("Pathfinder API listening on %s", addr)
	r.Run(addr)
}

type exportData struct {
	Goals      []storage.Goal      `json:"goals"`
	Plans      []storage.DailyPlan `json:"plans"`
	Events     []storage.Event     `json:"events"`
	CheckIns   []storage.CheckIn   `json:"check_ins"`
}

func exportHandler(c *gin.Context) {
	const uid = "local"
	var data exportData

	storage.DB.Where("user_id = ?", uid).Preload("Attachments").Find(&data.Goals)
	storage.DB.Where("user_id = ?", uid).Preload("Tasks").Find(&data.Plans)
	storage.DB.Where("user_id = ?", uid).Preload("Attachments").Find(&data.Events)
	storage.DB.Where("user_id = ?", uid).Find(&data.CheckIns)

	c.Header("Content-Disposition", "attachment; filename=pathfinder-export.json")
	c.JSON(http.StatusOK, data)
}

func importHandler(c *gin.Context) {
	var data exportData
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := json.Unmarshal(body, &data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for i := range data.Goals {
		data.Goals[i].UserID = "local"
		storage.DB.Save(&data.Goals[i])
	}
	for i := range data.Plans {
		data.Plans[i].UserID = "local"
		storage.DB.Save(&data.Plans[i])
		for j := range data.Plans[i].Tasks {
			storage.DB.Save(&data.Plans[i].Tasks[j])
		}
	}
	for i := range data.Events {
		data.Events[i].UserID = "local"
		storage.DB.Save(&data.Events[i])
	}
	for i := range data.CheckIns {
		data.CheckIns[i].UserID = "local"
		storage.DB.Save(&data.CheckIns[i])
	}

	c.JSON(http.StatusOK, gin.H{"message": "import successful"})
}
