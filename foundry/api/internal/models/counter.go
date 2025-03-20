package models

import (
	"fmt"
	"time"
)

// IDCounter tracks the monotonically increasing IDs for each project/branch combination
type IDCounter struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Project   string    `gorm:"not null" json:"project"`
	Branch    string    `gorm:"default:''" json:"branch"`
	Counter   int       `gorm:"not null;default:0" json:"counter"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for the IDCounter model
func (IDCounter) TableName() string {
	return "id_counters"
}

// UniqueKey returns the unique key for this project-branch combination
func (c *IDCounter) UniqueKey() string {
	if c.Branch == "" {
		return c.Project
	}
	return c.Project + "-" + c.Branch
}

// GetNextID returns the next ID for this counter in the format Project-Branch-XXX or Project-XXX
func (c *IDCounter) GetNextID() string {
	c.Counter++

	if c.Branch == "" {
		return c.Project + "-" + formatCounter(c.Counter)
	}
	return c.Project + "-" + c.Branch + "-" + formatCounter(c.Counter)
}

// formatCounter formats the counter as a 3-digit string (e.g., 1 -> "001")
func formatCounter(counter int) string {
	return fmt.Sprintf("%03d", counter)
}
