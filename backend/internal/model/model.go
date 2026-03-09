package model

import "time"

type User struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	FirstName     string    `gorm:"size:128;not null" json:"first_name"`
	LastName      string    `gorm:"size:128;not null" json:"last_name"`
	MiddleName    string    `gorm:"size:128" json:"middle_name"`
	FullName      string    `gorm:"size:255;not null" json:"full_name"`
	Username      string    `gorm:"size:64;not null;uniqueIndex" json:"username"`
	PasswordPlain string    `gorm:"size:255;not null" json:"password_plain"`
	Role          string    `gorm:"size:32;not null;index" json:"role"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Plan struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Year      int       `gorm:"not null;uniqueIndex" json:"year"`
	Status    string    `gorm:"size:32;not null;index" json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Task struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	PlanID            uint       `gorm:"not null;index" json:"plan_id"`
	ParentID          *uint      `gorm:"index" json:"parent_id,omitempty"`
	Title             string     `gorm:"size:255;not null" json:"title"`
	Description       string     `gorm:"type:text" json:"description"`
	PlannedValue      string     `gorm:"size:64" json:"planned_value"`
	Deadline          *time.Time `json:"deadline"`
	ResponsibleUserID uint       `gorm:"not null;index" json:"responsible_user_id"`
	Status            string     `gorm:"size:32;not null;index" json:"status"`
	ResultText        string     `gorm:"type:text" json:"result_text"`
	CompletionPercent int        `gorm:"default:0" json:"completion_percent"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type Report struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TaskID     uint      `gorm:"not null;index" json:"task_id"`
	FilePath   string    `gorm:"size:1024;not null" json:"file_path"`
	UploadedBy uint      `gorm:"not null;index" json:"uploaded_by"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type TaskLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TaskID    uint      `gorm:"not null;index" json:"task_id"`
	Action    string    `gorm:"size:64;not null" json:"action"`
	OldStatus string    `gorm:"size:32" json:"old_status"`
	NewStatus string    `gorm:"size:32" json:"new_status"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type PlanningPeriodIndicator struct {
	ID              uint               `gorm:"primaryKey" json:"id"`
	TargetIndicator string             `gorm:"type:text;not null" json:"target_indicator"`
	Unit            string             `gorm:"size:32;not null" json:"unit"`
	YearValues      map[string]float64 `gorm:"type:jsonb;serializer:json;not null" json:"year_values"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

func (PlanningPeriodIndicator) TableName() string {
	return "planning_period_indicators"
}
