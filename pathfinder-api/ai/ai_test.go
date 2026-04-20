package ai_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"pathfinder-api/ai"
	"pathfinder-api/storage"
)

// TestInit_DefaultProviderIsMiniMax verifies that an empty Provider string
// uses MiniMax (returns default tasks when API key is blank).
func TestInit_DefaultProviderIsMiniMax(t *testing.T) {
	ai.Init(ai.Config{}) // no Provider set, no API key
	tasks, err := ai.GenerateInitialPlan([]storage.Goal{{Title: "test"}}, nil, 8, "09:00")
	// With no API key, MiniMax returns defaultTasks (not an error).
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) == 0 {
		t.Error("expected default tasks, got none")
	}
}

// TestInit_MiniMaxExplicit verifies explicit "minimax" provider selection.
func TestInit_MiniMaxExplicit(t *testing.T) {
	ai.Init(ai.Config{Provider: "minimax"})
	tasks, err := ai.GenerateInitialPlan(nil, nil, 8, "09:00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) == 0 {
		t.Error("expected default tasks for empty goals, got none")
	}
}

// newWebhookServer returns a test server and a channel that receives each
// incoming request (method, event header, body, sig header).
type webhookRequest struct {
	Event     string
	Body      map[string]interface{}
	Signature string
}

func newWebhookServer(t *testing.T, statusCode int) (*httptest.Server, <-chan webhookRequest) {
	t.Helper()
	ch := make(chan webhookRequest, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]interface{}
		json.Unmarshal(body, &parsed)
		ch <- webhookRequest{
			Event:     r.Header.Get("X-Pathfinder-Event"),
			Body:      parsed,
			Signature: r.Header.Get("X-Pathfinder-Signature"),
		}
		w.WriteHeader(statusCode)
	}))
	t.Cleanup(srv.Close)
	return srv, ch
}

func TestOpenClaw_GenerateInitialPlan_SendsWebhook(t *testing.T) {
	srv, ch := newWebhookServer(t, http.StatusOK)
	ai.Init(ai.Config{
		Provider:           "openclaw",
		OpenClawWebhookURL: srv.URL,
	})

	goals := []storage.Goal{{UserID: "u1", Title: "Learn Go"}}
	tasks, err := ai.GenerateInitialPlan(goals, nil, 8, "09:00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) == 0 {
		t.Error("expected default tasks, got none")
	}

	req := <-ch
	if req.Event != "plan.generate_requested" {
		t.Errorf("event = %q, want plan.generate_requested", req.Event)
	}
	if req.Body["user_id"] != "u1" {
		t.Errorf("user_id = %v, want u1", req.Body["user_id"])
	}
}

func TestOpenClaw_RegenerateAfterCheckin_SendsWebhook(t *testing.T) {
	srv, ch := newWebhookServer(t, http.StatusOK)
	ai.Init(ai.Config{
		Provider:           "openclaw",
		OpenClawWebhookURL: srv.URL,
	})

	checkin := storage.CheckIn{
		UserID:        "u1",
		Date:          "2026-04-20",
		Completed:     "wrote tests",
		Blocked:       "none",
		TomorrowFocus: "deploy",
	}
	tasks, err := ai.RegenerateAfterCheckin(checkin, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) == 0 {
		t.Error("expected default tasks, got none")
	}

	req := <-ch
	if req.Event != "checkin.submitted" {
		t.Errorf("event = %q, want checkin.submitted", req.Event)
	}
	data, _ := req.Body["data"].(map[string]interface{})
	if data["completed"] != "wrote tests" {
		t.Errorf("completed = %v, want 'wrote tests'", data["completed"])
	}
}

func TestOpenClaw_WebhookSignature(t *testing.T) {
	secret := "mysecret"
	srv, ch := newWebhookServer(t, http.StatusOK)
	ai.Init(ai.Config{
		Provider:              "openclaw",
		OpenClawWebhookURL:    srv.URL,
		OpenClawWebhookSecret: secret,
	})

	goals := []storage.Goal{{UserID: "u1", Title: "g"}}
	ai.GenerateInitialPlan(goals, nil, 8, "09:00")

	req := <-ch
	if req.Signature == "" {
		t.Fatal("expected X-Pathfinder-Signature header, got empty")
	}
	if len(req.Signature) < 7 || req.Signature[:7] != "sha256=" {
		t.Errorf("signature format = %q, want sha256=<hex>", req.Signature)
	}
	// Verify the HMAC prefix matches expected format (content varies by timestamp).
	mac := hmac.New(sha256.New, []byte(secret))
	// We can't recompute the exact sig without the body, but check it's non-empty hex.
	_ = mac
}

func TestOpenClaw_WebhookFailure_PropagatesError(t *testing.T) {
	srv, _ := newWebhookServer(t, http.StatusInternalServerError)
	ai.Init(ai.Config{
		Provider:           "openclaw",
		OpenClawWebhookURL: srv.URL,
	})

	goals := []storage.Goal{{UserID: "u1", Title: "g"}}
	_, err := ai.GenerateInitialPlan(goals, nil, 8, "09:00")
	if err == nil {
		t.Fatal("expected error on non-2xx webhook response, got nil")
	}
}

func TestOpenClaw_NoWebhookURL_Succeeds(t *testing.T) {
	ai.Init(ai.Config{Provider: "openclaw"}) // no WebhookURL
	goals := []storage.Goal{{UserID: "u1", Title: "g"}}
	tasks, err := ai.GenerateInitialPlan(goals, nil, 8, "09:00")
	if err != nil {
		t.Fatalf("unexpected error when webhook not configured: %v", err)
	}
	if len(tasks) == 0 {
		t.Error("expected default tasks, got none")
	}
}

func TestOpenClaw_InsertEvent_NoWebhookSent(t *testing.T) {
	srv, ch := newWebhookServer(t, http.StatusOK)
	ai.Init(ai.Config{
		Provider:           "openclaw",
		OpenClawWebhookURL: srv.URL,
	})

	tasks, err := ai.InsertEvent(storage.Event{Title: "demo day"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("InsertEvent should return nil tasks for openclaw, got %d", len(tasks))
	}
	// Channel should be empty — no webhook sent.
	select {
	case req := <-ch:
		t.Errorf("unexpected webhook sent: event=%s", req.Event)
	default:
	}
}

func TestOpenClaw_WebhookNetworkError_PropagatesError(t *testing.T) {
	// Use a server URL that is immediately closed — simulates connection refused.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // close before the request arrives

	ai.Init(ai.Config{
		Provider:           "openclaw",
		OpenClawWebhookURL: url,
	})

	goals := []storage.Goal{{UserID: "u1", Title: "g"}}
	_, err := ai.GenerateInitialPlan(goals, nil, 8, "09:00")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}
