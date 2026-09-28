package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

const testBotToken = "test-bot-token"

func TestValidateInitData(t *testing.T) {
	initData := signedInitData(t, testBotToken, map[string]string{
		"query_id": "query-1",
		"user":     `{"id":12345,"username":"tester","name":"Test User"}`,
	})

	if err := ValidateInitData(initData, testBotToken); err != nil {
		t.Fatalf("ValidateInitData() error = %v", err)
	}
	if err := ValidateInitData(initData, "wrong-token"); err == nil {
		t.Fatal("ValidateInitData() accepted invalid signature")
	}
}

func TestParseUserUsesMaxID(t *testing.T) {
	initData := signedInitData(t, testBotToken, map[string]string{
		"user": `{"id":12345,"username":"tester","name":"Test User"}`,
	})

	user, err := ParseUser(initData)
	if err != nil {
		t.Fatalf("ParseUser() error = %v", err)
	}
	if user.UserID != 12345 {
		t.Fatalf("UserID = %d, want 12345", user.UserID)
	}
	if user.DisplayName != "Test User" {
		t.Fatalf("DisplayName = %q, want %q", user.DisplayName, "Test User")
	}
}

func TestParseUserRejectsMissingID(t *testing.T) {
	initData := signedInitData(t, testBotToken, map[string]string{
		"user": `{"username":"tester"}`,
	})

	if _, err := ParseUser(initData); err == nil {
		t.Fatal("ParseUser() accepted missing id")
	}
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DEV_MODE", "1")

	valid := signedInitData(t, testBotToken, map[string]string{
		"user": `{"id":777,"username":"tester","name":"Tester"}`,
	})

	tests := []struct {
		name       string
		host       string
		initData   string
		wantStatus int
		wantUser   int64
	}{
		{name: "signed user", host: "example.com", initData: valid, wantStatus: 200, wantUser: 777},
		{name: "missing data fails closed", host: "example.com", wantStatus: 401},
		{name: "invalid data fails closed", host: "example.com", initData: "hash=bad", wantStatus: 401},
		{name: "development host", host: "localhost:9300", wantStatus: 200, wantUser: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(AuthMiddleware(testBotToken))
			router.GET("/private", func(c *gin.Context) {
				c.JSON(200, gin.H{"user_id": getContextUserID(c)})
			})

			req := httptest.NewRequest("GET", "http://"+tt.host+"/private", nil)
			if tt.initData != "" {
				req.Header.Set("X-Max-InitData", tt.initData)
			}
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)

			if res.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", res.Code, tt.wantStatus, res.Body.String())
			}
			if tt.wantStatus == 200 && !strings.Contains(res.Body.String(), `"user_id":`+itoa(tt.wantUser)) {
				t.Fatalf("body = %s, want user_id %d", res.Body.String(), tt.wantUser)
			}
		})
	}
}

func TestDevelopmentBypassRequiresEnvironment(t *testing.T) {
	previous, existed := os.LookupEnv("DEV_MODE")
	if existed {
		t.Cleanup(func() { _ = os.Setenv("DEV_MODE", previous) })
	} else {
		t.Cleanup(func() { _ = os.Unsetenv("DEV_MODE") })
	}
	_ = os.Unsetenv("DEV_MODE")

	if developmentRequest("localhost:9300") {
		t.Fatal("development bypass enabled without DEV_MODE")
	}
}

func signedInitData(t *testing.T, token string, values map[string]string) string {
	t.Helper()
	params := url.Values{}
	for key, value := range values {
		params.Set(key, value)
	}
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params.Get(key))
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secret.Write([]byte(token))
	computed := hmac.New(sha256.New, secret.Sum(nil))
	_, _ = computed.Write([]byte(strings.Join(parts, "\n")))
	params.Set("hash", hex.EncodeToString(computed.Sum(nil)))
	return params.Encode()
}

func getContextUserID(c *gin.Context) int64 {
	value, _ := c.Get("user_id")
	id, _ := value.(int64)
	return id
}

func itoa(value int64) string {
	if value == 0 {
		return "0"
	}
	digits := ""
	for value > 0 {
		digits = string(rune('0'+value%10)) + digits
		value /= 10
	}
	return digits
}
