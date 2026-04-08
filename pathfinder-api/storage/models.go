package storage

import "time"

type User struct {
	ID                  uint      `gorm:"primarykey" json:"id"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	Username            string    `json:"username" gorm:"uniqueIndex"`
	Email               string    `json:"email" gorm:"uniqueIndex"`
	Password            string    `json:"-"`
	Status              string    `json:"status" gorm:"default:pending"` // pending | active
	VerificationToken   string    `json:"-"`
	TokenExpiresAt      time.Time `json:"-"`
	ResetToken          string    `json:"-"`
	ResetTokenExpiresAt time.Time `json:"-"`
}

type UserProfile struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	UserID         string    `json:"user_id" gorm:"uniqueIndex"`
	Bio            string    `json:"bio"`
	ResumeFilename string    `json:"resume_filename"`
	ResumeMimeType string    `json:"resume_mime_type"`
	ResumeData     []byte    `json:"-"`
}

type Goal struct {
	ID          uint             `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	UserID      string           `json:"user_id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Type        string           `json:"type"`   // primary | secondary
	Status      string           `json:"status"` // active | paused | completed
	Timeline    string           `json:"timeline"`
	Attachments []GoalAttachment `gorm:"foreignKey:GoalID" json:"attachments,omitempty"`
}

type GoalAttachment struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	GoalID     uint      `json:"goal_id"`
	Filename   string    `json:"filename"`
	MimeType   string    `json:"mime_type"`
	Data       []byte    `json:"-"`
	DataBase64 string    `gorm:"-" json:"data,omitempty"`
}

type DailyPlan struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID    string    `json:"user_id"`
	Date      string    `json:"date"` // YYYY-MM-DD
	Tasks     []Task    `gorm:"foreignKey:PlanID" json:"tasks,omitempty"`
}

type Task struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	PlanID         uint      `json:"plan_id"`
	GoalID         *uint     `json:"goal_id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Status         string    `json:"status"` // pending | done | skipped
	SortOrder      int       `json:"sort_order"`
	SuggestedStart string    `json:"suggested_start"` // HH:MM
	SuggestedEnd   string    `json:"suggested_end"`   // HH:MM
}

type Event struct {
	ID          uint              `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	UserID      string            `json:"user_id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	EventDate   string            `json:"event_date"` // YYYY-MM-DD
	Status      string            `json:"status"`     // upcoming | completed
	RetroNote   string            `json:"retro_note"`
	Attachments []EventAttachment `gorm:"foreignKey:EventID" json:"attachments,omitempty"`
}

type EventAttachment struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	EventID   uint      `json:"event_id"`
	Filename  string    `json:"filename"`
	MimeType  string    `json:"mime_type"`
	Data      []byte    `json:"-"`
}

type CheckIn struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	UserID        string    `json:"user_id"`
	Date          string    `json:"date"` // YYYY-MM-DD
	Completed     string    `json:"completed"`
	Blocked       string    `json:"blocked"`
	TomorrowFocus string    `json:"tomorrow_focus"`
}
