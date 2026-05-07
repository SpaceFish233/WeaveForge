package coordinator

import "time"

// Notification from an agent analysis, pushed to frontend via Wails events.
type Notification struct {
	ID        string `json:"id"`
	Agent     string `json:"agent"`     // consistency / style / foreshadow / inspiration
	Title     string `json:"title"`
	Content   string `json:"content"`
	Severity  string `json:"severity"`  // info / warning / success
	Action    string `json:"action"`    // accept / dismiss
	SessionID string `json:"session_id"`
	Time      string `json:"time"`
}

// SessionEvent records one interaction for session history.
type SessionEvent struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Agent      string    `json:"agent"`
	Content    string    `json:"content"`
	UserAction string    `json:"user_action"` // accepted / dismissed / ignored
	Timestamp  time.Time `json:"timestamp"`
}
