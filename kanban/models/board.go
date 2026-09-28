package models

import "time"

type Board struct {
	ID          int64     `json:"id"`
	OrgID       int64     `json:"org_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsArchived  bool      `json:"is_archived"`
	CreatedBy   int64     `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

type BoardDetail struct {
	Board
	Columns []Column    `json:"columns"`
	Members []OrgMember `json:"members"`
}

type BoardCreate struct {
	Name        string         `json:"name" binding:"required,min=1,max=120"`
	Description string         `json:"description" binding:"max=2000"`
	Columns     []ColumnCreate `json:"columns" binding:"dive"`
}

type BoardUpdate struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=1,max=120"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=2000"`
	Archived    *bool   `json:"is_archived,omitempty"`
}
