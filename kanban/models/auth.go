package models

type AuthRequest struct {
	InitData string `json:"initData" binding:"required"`
}

type AuthUser struct {
	UserID      int64  `json:"user_id"`
	Username    string `json:"username,omitempty"`
	FirstName   string `json:"first_name,omitempty"`
	LastName    string `json:"last_name,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

type AuthResponse struct {
	OK    bool     `json:"ok"`
	User  AuthUser `json:"user"`
	Error string   `json:"error,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
