package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

type Task struct {
	ID          int64      `json:"id"`
	BoardID     int64      `json:"board_id"`
	ColumnID    int64      `json:"column_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Position    int        `json:"position"`
	Priority    string     `json:"priority"`
	Deadline    *string    `json:"deadline"`
	CreatedBy   int64      `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Assignees   []Assignee `json:"assignees"`
}

type TaskDetail struct {
	Task
	Checklists []Checklist `json:"checklists"`
	Comments   []Comment   `json:"comments"`
}

type TaskCreate struct {
	Title       string  `json:"title" binding:"required,min=1,max=300"`
	Description string  `json:"description" binding:"max=10000"`
	ColumnID    int64   `json:"column_id" binding:"omitempty,gt=0"`
	Priority    string  `json:"priority" binding:"omitempty,oneof=low medium high urgent"`
	Deadline    *string `json:"deadline"`
}

type TaskUpdate struct {
	Title       *string      `json:"title,omitempty" binding:"omitempty,min=1,max=300"`
	Description *string      `json:"description,omitempty" binding:"omitempty,max=10000"`
	Priority    *string      `json:"priority,omitempty" binding:"omitempty,oneof=low medium high urgent"`
	Deadline    NullableDate `json:"deadline"`
}

type TaskMove struct {
	ColumnID int64 `json:"column_id" binding:"required,gt=0"`
	Position *int  `json:"position" binding:"required,gte=0"`
}

type NullableDate struct {
	Present bool
	Valid   bool
	Value   string
}

func (d *NullableDate) UnmarshalJSON(data []byte) error {
	d.Present = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		d.Valid = false
		d.Value = ""
		return nil
	}
	if err := json.Unmarshal(data, &d.Value); err != nil {
		return err
	}
	if d.Value == "" {
		d.Valid = false
		d.Value = ""
		return nil
	}
	if _, err := time.Parse("2006-01-02", d.Value); err != nil {
		return fmt.Errorf("deadline must use YYYY-MM-DD")
	}
	d.Valid = true
	return nil
}

type Assignee struct {
	TaskID      int64  `json:"task_id"`
	UserID      int64  `json:"user_id"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}
