package contracts

// ChangeItem یک تغییر دلتا برای کلاینت آفلاین.
type ChangeItem struct {
	Entity  string         `json:"entity"`
	Op      string         `json:"op"`
	ID      string         `json:"id"`
	Version int            `json:"version"`
	Payload map[string]any `json:"payload,omitempty"`
}

// ChangeSet نتیجهٔ درخواست دلتا از نشانگر since.
type ChangeSet struct {
	Since   int          `json:"since"`
	NextSeq int          `json:"nextSeq"`
	Changes []ChangeItem `json:"changes"`
}

// Mutation عملیات آفلاین کلاینت؛ opId برای ایدمپوتنت بودن الزامی است.
type Mutation struct {
	OpID        string         `json:"opId"`
	Entity      string         `json:"entity"`
	Op          string         `json:"op"`
	ID          string         `json:"id"`
	Payload     map[string]any `json:"payload,omitempty"`
	ClientTs    string         `json:"clientTs,omitempty"`
	BaseVersion int            `json:"baseVersion,omitempty"`
}

// RejectedMutation دلیل رد یک عملیات در سرور.
type RejectedMutation struct {
	OpID          string `json:"opId"`
	Reason        string `json:"reason"`
	ServerVersion int    `json:"serverVersion,omitempty"`
	Message       string `json:"message,omitempty"`
}

// MutationResult نتیجهٔ اعمال دسته‌ای عملیات.
type MutationResult struct {
	Accepted []string           `json:"accepted"`
	Rejected []RejectedMutation `json:"rejected"`
	NextSeq  int                `json:"nextSeq"`
}
