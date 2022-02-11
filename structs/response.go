package structs

type MismatchQueueResponse struct {
	TaskId uint   `json:"task_id" form:"task_id"`
	Queue  string `json:"queue" form:"queue"`
}
