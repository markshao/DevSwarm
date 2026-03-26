package notification

import "time"

const (
	StateRunning        = "running"
	StateQuietCandidate = "quiet_candidate"
	StateWaitingInput   = "waiting_input"
	StateCompletedIdle  = "completed_idle"
	StateUnknown        = "unknown"
	StateMissing        = "missing"
)

type ServiceConfig struct {
	Enabled             bool
	Provider            string
	PollInterval        time.Duration
	SilenceThreshold    time.Duration
	ReminderInterval    time.Duration
	SimilarityThreshold float64
	TailLines           int
	LLMEnabled          bool
	LastBlock           LastBlockConfig
	Lark                LarkConfig
}

type LastBlockConfig struct {
	Enabled  bool
	Mode     string
	Prefix   string
	Regex    string
	MaxChars int
}

type LarkConfig struct {
	AppID     string
	AppSecret string
	BaseURL   string
	OpenID    string
	ChatID    string
	UrgentApp bool
	CardTitle string
}

type Watcher struct {
	NodeName             string    `json:"node_name"`
	Label                string    `json:"label,omitempty"`
	SessionName          string    `json:"session_name"`
	PaneID               string    `json:"pane_id"`
	RegisteredAt         time.Time `json:"registered_at"`
	State                string    `json:"state"`
	StateEnteredAt       time.Time `json:"state_entered_at,omitempty"`
	StableSince          time.Time `json:"stable_since,omitempty"`
	LastReason           string    `json:"last_reason,omitempty"`
	LastHash             string    `json:"last_hash,omitempty"`
	LastNormalizedScreen string    `json:"last_normalized_screen,omitempty"`
	LastSimilarity       float64   `json:"last_similarity,omitempty"`
	LastChangeAt         time.Time `json:"last_change_at,omitempty"`
	LastObservedAt       time.Time `json:"last_observed_at,omitempty"`
	LastClassifiedHash   string    `json:"last_classified_hash,omitempty"`
	LastClassifiedState  string    `json:"last_classified_state,omitempty"`
	LastClassifiedAt     time.Time `json:"last_classified_at,omitempty"`
	LastLLMReason        string    `json:"last_llm_reason,omitempty"`
	WaitEventID          int       `json:"wait_event_id,omitempty"`
	AckedWaitEventID     int       `json:"acked_wait_event_id,omitempty"`
	MutedWaitEventID     int       `json:"muted_wait_event_id,omitempty"`
	LastAgentBlock       string    `json:"last_agent_block,omitempty"`
	LastAgentBlockAt     time.Time `json:"last_agent_block_at,omitempty"`
	LastNotifyAt         time.Time `json:"last_notify_at,omitempty"`
	NotifyCount          int       `json:"notify_count,omitempty"`
	LastError            string    `json:"last_error,omitempty"`
}

type Registry struct {
	Watchers map[string]*Watcher `json:"watchers"`
}

type ServiceStatus struct {
	PID        int       `json:"pid"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	LastLoopAt time.Time `json:"last_loop_at,omitempty"`
	LastError  string    `json:"last_error,omitempty"`
}

type Classification struct {
	State  string `json:"state"`
	Reason string `json:"reason"`
}

func HasPendingWaitEvent(watcher *Watcher) bool {
	if watcher == nil {
		return false
	}
	return watcher.WaitEventID > watcher.AckedWaitEventID
}
