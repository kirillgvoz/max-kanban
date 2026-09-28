package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {
	userID := flag.Int64("user-id", 0, "Test user ID")
	username := flag.String("username", "", "Test username")
	displayName := flag.String("display-name", "", "Test display name")
	flag.Parse()
	token := os.Getenv("MAX_BOT_TOKEN")
	initData, err := buildInitData(token, *userID, *username, *displayName, time.Now().Unix())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Println(initData)
}

func buildInitData(token string, userID int64, username, displayName string, issuedAt int64) (string, error) {
	if token == "" {
		return "", fmt.Errorf("MAX_BOT_TOKEN is required")
	}
	if userID <= 0 {
		return "", fmt.Errorf("user-id must be positive")
	}
	user, err := json.Marshal(map[string]any{
		"id":            userID,
		"username":      username,
		"first_name":    displayName,
		"last_name":     "",
		"language_code": "ru",
	})
	if err != nil {
		return "", err
	}
	values := url.Values{}
	values.Set("auth_date", fmt.Sprintf("%d", issuedAt))
	values.Set("query_id", fmt.Sprintf("api-check-%d", userID))
	values.Set("user", string(user))
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(token))
	check := hmac.New(sha256.New, secret.Sum(nil))
	check.Write([]byte(strings.Join(parts, "\n")))
	values.Set("hash", hex.EncodeToString(check.Sum(nil)))
	return values.Encode(), nil
}
