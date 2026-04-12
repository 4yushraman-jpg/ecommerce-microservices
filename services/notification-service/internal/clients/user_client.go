package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserServiceClient struct {
	BaseURL    string
	HTTPClient *http.Client
	JWTSecret  []byte
}

func NewUserServiceClient(baseURL string, jwtSecret []byte) *UserServiceClient {
	return &UserServiceClient{
		BaseURL: baseURL,
		// 5-second timeout is crucial so a hanging User Service doesn't crash the Notification Service
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		JWTSecret:  jwtSecret,
	}
}

// We only define the exact fields we need from the User Service's JSON response
type userResponse struct {
	Email string `json:"email"`
}

func (c *UserServiceClient) GetUserEmail(ctx context.Context, userID int64) (string, error) {
	url := fmt.Sprintf("%s/users/%d", c.BaseURL, userID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	// Sign the request as an Internal Admin
	token, _ := c.generateInternalAdminToken()
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error calling user-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("user-service returned status %d", resp.StatusCode)
	}

	var user userResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("failed to decode user response: %w", err)
	}

	if user.Email == "" {
		return "", fmt.Errorf("user-service returned empty email for user %d", userID)
	}

	return user.Email, nil
}

// Generates a 1-minute system token so this request bypasses normal user auth
func (c *UserServiceClient) generateInternalAdminToken() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 0, // System User ID
		"role":    "admin",
		"exp":     time.Now().Add(time.Minute * 1).Unix(),
	})
	return token.SignedString(c.JWTSecret)
}
