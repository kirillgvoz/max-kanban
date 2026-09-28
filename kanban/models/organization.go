package models

import "time"

type Organization struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type OrgMember struct {
	OrgID       int64     `json:"org_id"`
	UserID      int64     `json:"user_id"`
	Username    string    `json:"username,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	Role        string    `json:"role"`
	JoinedAt    time.Time `json:"joined_at"`
}

type OrgDetail struct {
	Organization
	Members []OrgMember `json:"members"`
}

type OrgCreate struct {
	Name string `json:"name" binding:"required,min=2,max=80"`
}

type OrgUpdate struct {
	Name      *string `json:"name,omitempty" binding:"omitempty,min=2,max=80"`
	AvatarURL *string `json:"avatar_url,omitempty" binding:"omitempty,max=500"`
}

type MemberCreate struct {
	UserID   int64  `json:"user_id" binding:"required,gt=0"`
	Username string `json:"username" binding:"max=100"`
	Name     string `json:"display_name" binding:"max=150"`
	Role     string `json:"role" binding:"omitempty,oneof=admin member"`
}
