package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"kanban/models"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(botToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.TrimSpace(botToken) == "" {
			c.AbortWithStatusJSON(500, models.ErrorResponse{Error: "Authentication is not configured"})
			return
		}

		initData := c.GetHeader("X-Max-InitData")
		if initData == "" {
			initData = c.Query("initData")
		}

		if initData == "" {
			if developmentRequest(c.Request.Host) {
				setDevelopmentUser(c)
				c.Next()
				return
			}
			c.AbortWithStatusJSON(401, models.ErrorResponse{Error: "Missing initData"})
			return
		}

		if err := ValidateInitData(initData, botToken); err != nil {
			c.AbortWithStatusJSON(401, models.ErrorResponse{Error: "Invalid initData"})
			return
		}

		user, err := ParseUser(initData)
		if err != nil {
			c.AbortWithStatusJSON(401, models.ErrorResponse{Error: "Invalid user data"})
			return
		}

		c.Set("user_id", user.UserID)
		c.Set("username", user.Username)
		c.Set("display_name", user.DisplayName)
		c.Next()
	}
}

func ValidateInitData(initData, botToken string) error {
	params, err := url.ParseQuery(initData)
	if err != nil {
		return fmt.Errorf("parse init data: %w", err)
	}

	hash := params.Get("hash")
	if hash == "" {
		return fmt.Errorf("missing hash")
	}

	keys := make([]string, 0, len(params))
	for key := range params {
		if key != "hash" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params.Get(key))
	}

	secret := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secret.Write([]byte(botToken))
	computed := hmac.New(sha256.New, secret.Sum(nil))
	_, _ = computed.Write([]byte(strings.Join(parts, "\n")))
	actual := hex.EncodeToString(computed.Sum(nil))

	if !hmac.Equal([]byte(actual), []byte(hash)) {
		return fmt.Errorf("hash mismatch")
	}
	return nil
}

func ParseUser(initData string) (*models.AuthUser, error) {
	params, err := url.ParseQuery(initData)
	if err != nil {
		return nil, err
	}

	var raw struct {
		ID        int64  `json:"id"`
		UserID    int64  `json:"user_id"`
		Username  string `json:"username"`
		Name      string `json:"name"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if err := json.Unmarshal([]byte(params.Get("user")), &raw); err != nil {
		return nil, fmt.Errorf("decode user: %w", err)
	}

	userID := raw.ID
	if userID == 0 {
		userID = raw.UserID
	}
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}

	firstName := strings.TrimSpace(raw.FirstName)
	lastName := strings.TrimSpace(raw.LastName)
	displayName := strings.TrimSpace(strings.Join([]string{firstName, lastName}, " "))
	if displayName == "" {
		displayName = strings.TrimSpace(raw.Name)
	}
	if displayName == "" {
		displayName = strings.TrimSpace(raw.Username)
	}

	return &models.AuthUser{
		UserID:      userID,
		Username:    strings.TrimSpace(raw.Username),
		FirstName:   firstName,
		LastName:    lastName,
		DisplayName: displayName,
	}, nil
}

func developmentRequest(host string) bool {
	if os.Getenv("DEV_MODE") != "1" {
		return false
	}
	hostname, _, err := net.SplitHostPort(host)
	if err != nil {
		hostname = host
	}
	return hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1"
}

func setDevelopmentUser(c *gin.Context) {
	userID := int64(1)
	if raw := os.Getenv("DEV_USER_ID"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			userID = parsed
		}
	}
	c.Set("user_id", userID)
	c.Set("username", "dev")
	c.Set("display_name", "Dev User")
}
