package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	clientID      = "Iv23likqjTMtSXzFRBNi"
	deviceCodeURL = "https://github.com/login/device/code"
	tokenURL      = "https://github.com/login/oauth/access_token"
	scope         = "read:org read:user"
)

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
}

type Credentials struct {
	Token     string `json:"token"`
	Username  string `json:"username"`
	GeminiKey string `json:"gemini_key,omitempty"`
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "dsc"), nil
}

func credentialsPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.json"), nil
}

func SaveCredentials(creds Credentials) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path, err := credentialsPath()
	if err != nil {
		return err
	}
	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func LoadCredentials() (Credentials, error) {
	path, err := credentialsPath()
	if err != nil {
		return Credentials{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Credentials{}, err
	}
	var creds Credentials
	return creds, json.Unmarshal(data, &creds)
}

func RemoveCredentials() error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}
	return os.Remove(path)
}

func RequestDeviceCode() (DeviceCodeResponse, error) {
	req, err := http.NewRequest("POST", deviceCodeURL, strings.NewReader(url.Values{
		"client_id": {clientID},
		"scope":     {scope},
	}.Encode()))
	if err != nil {
		return DeviceCodeResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return DeviceCodeResponse{}, err
	}
	defer resp.Body.Close()

	var result DeviceCodeResponse
	return result, json.NewDecoder(resp.Body).Decode(&result)
}

func PollForToken(deviceCode string, interval int) (string, error) {
	wait := time.Duration(interval) * time.Second

	for {
		time.Sleep(wait)

		req, err := http.NewRequest("POST", tokenURL, strings.NewReader(url.Values{
			"client_id":   {clientID},
			"device_code": {deviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		}.Encode()))
		if err != nil {
			return "", err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}

		var body struct {
			AccessToken string `json:"access_token"`
			Error       string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			resp.Body.Close()
			return "", err
		}
		resp.Body.Close()

		switch body.Error {
		case "":
			if body.AccessToken != "" {
				return body.AccessToken, nil
			}
		case "authorization_pending":
			continue
		case "slow_down":
			wait += 5 * time.Second
		case "expired_token":
			return "", fmt.Errorf("code expired, run dsc auth login again")
		case "access_denied":
			return "", fmt.Errorf("access denied")
		}
	}
}

func VerifyOrgMembership(token string) (string, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var user struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", err
	}

	// Use authenticated user membership endpoint (works for private memberships)
	req2, _ := http.NewRequest("GET",
		"https://api.github.com/user/memberships/orgs/"+Org, nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.Header.Set("Accept", "application/vnd.github+json")

	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		return "", err
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != 200 {
		return "", fmt.Errorf("@%s is not a member of %s", user.Login, Org)
	}

	var membership struct {
		State string `json:"state"`
	}
	json.NewDecoder(resp2.Body).Decode(&membership)

	if membership.State != "active" {
		return "", fmt.Errorf("@%s membership in %s is not active", user.Login, Org)
	}

	return user.Login, nil
}

// GHToken tries to reuse the existing gh CLI token for 0-friction login
func GHToken() (string, error) {
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func IsAuthenticated() bool {
	creds, err := LoadCredentials()
	if err != nil {
		return false
	}
	return strings.TrimSpace(creds.Token) != ""
}
