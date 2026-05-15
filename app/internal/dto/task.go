package dto

// CreateTaskRequest is the request body for creating a task.
type CreateTaskRequest struct {
	Type    string `json:"type" binding:"required"`
	Payload string `json:"payload"`
}

// TaskResponse is the task data returned in API responses.
type TaskResponse struct {
	ID           string  `json:"id"`
	TaskID       string  `json:"task_id"`
	Type         string  `json:"type"`
	Payload      string  `json:"payload"`
	Status       string  `json:"status"`
	RetryCount   int     `json:"retry_count"`
	MaxRetries   int     `json:"max_retries"`
	Result       string  `json:"result"`
	ErrorMessage string  `json:"error_message"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
	FinishedAt   *string `json:"finished_at"`
}
