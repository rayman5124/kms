package dto

type ErrRes struct {
	Status    int    `json:"status"`
	Timestamp string `json:"timestamp"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Message   string `json:"message"`
}
