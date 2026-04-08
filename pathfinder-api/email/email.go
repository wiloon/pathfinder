package email

import (
"bytes"
"encoding/json"
"fmt"
"log"
"net/http"
)

var (
apiKey  string
fromAddr string
frontendBaseURL string
)

func Init(key, from, baseURL string) {
	apiKey = key
	fromAddr = from
	frontendBaseURL = baseURL
}

type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func sendEmail(to, subject, htmlBody string) error {
	if apiKey == "" {
		log.Printf("email: resend api key not set, skipping email to %s", to)
		return nil
	}

	payload := resendRequest{
		From:    fromAddr,
		To:      []string{to},
		Subject: subject,
		HTML:    htmlBody,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal email payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create email request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send email request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("resend returned status %d", resp.StatusCode)
	}
	log.Printf("email: sent to %s, status %d", to, resp.StatusCode)
	return nil
}

// SendVerificationEmail sends an activation link to the user.
func SendVerificationEmail(to, username, token string) error {
	link := fmt.Sprintf("%s/verify-email?token=%s", frontendBaseURL, token)
	subject := "Activate your Pathfinder account"
	html := fmt.Sprintf(
"<p>Hi %s,</p><p>Click the link below to activate your Pathfinder account. The link expires in 48 hours.</p><p><a href=%q>Activate my account</a></p><p>If you did not create an account, you can safely ignore this email.</p>",
username, link,
)
	return sendEmail(to, subject, html)
}

// SendPasswordResetEmail sends a password reset link to the user.
func SendPasswordResetEmail(to, username, token string) error {
	link := fmt.Sprintf("%s/reset-password?token=%s", frontendBaseURL, token)
	subject := "Reset your Pathfinder password"
	html := fmt.Sprintf(
"<p>Hi %s,</p><p>Click the link below to reset your Pathfinder password. The link expires in 1 hour.</p><p><a href=%q>Reset my password</a></p><p>If you did not request a password reset, you can safely ignore this email.</p>",
username, link,
)
	return sendEmail(to, subject, html)
}
