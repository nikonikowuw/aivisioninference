package dto

// ScheduledTaskRequest 创建/更新定时任务请求
type ScheduledTaskRequest struct {
	Name           string `json:"name" binding:"required,min=1,max=128"`
	Command        string `json:"command" binding:"required,min=1"`
	CronExpr       string `json:"cron_expr" binding:"omitempty,min=1,max=100"`
	TimeoutSeconds int    `json:"timeout_seconds" binding:"omitempty,min=1,max=3600"`
}

// ScheduledTaskUpdateRequest 更新定时任务请求
type ScheduledTaskUpdateRequest struct {
	Name           *string `json:"name" binding:"omitempty,min=1,max=128"`
	Command        *string `json:"command" binding:"omitempty,min=1"`
	CronExpr       *string `json:"cron_expr" binding:"omitempty,min=1,max=100"`
	TimeoutSeconds *int    `json:"timeout_seconds" binding:"omitempty,min=1,max=3600"`
	Status         *string `json:"status" binding:"omitempty,oneof=active disabled"`
}

// ScheduledTaskListRequest 定时任务列表查询
type ScheduledTaskListRequest struct {
	PageRequest
	NodeID string `form:"node_id"`
	Status string `form:"status" binding:"omitempty,oneof=active disabled running"`
}

// TaskExecutionListRequest 任务执行记录列表查询
type TaskExecutionListRequest struct {
	PageRequest
	TaskID string `form:"task_id" binding:"required"`
	Status string `form:"status" binding:"omitempty,oneof=running success failed timeout"`
}

// TaskExecutionCallbackRequest 引擎执行结果回调
type TaskExecutionCallbackRequest struct {
	TaskID     string `json:"task_id"`
	Status     string `json:"status" binding:"required,oneof=success failed timeout"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
}

// WebSocketTerminalMessage WebSocket终端消息
type WebSocketTerminalMessage struct {
	Type      string `json:"type"`      // "input", "output", "resize", "close", "error", "ping"
	SessionID string `json:"session_id,omitempty"`
	Data      string `json:"data,omitempty"`       // terminal input/output data
	Cols      int    `json:"cols,omitempty"`       // resize columns
	Rows      int    `json:"rows,omitempty"`       // resize rows
	Error     string `json:"error,omitempty"`
}
