package structs

type IngredientQueueParam struct {
	Type      string `json:"type" form:"type"`
	MemberId  string `json:"member_id" form:"member_id"`
	TaskID    uint   `json:"task_id" form:"task_id"`
	Result    string `json:"result" form:"result"`
	IsDie     bool   `json:"is_die" form:"is_die"`
	QueueType string `json:"queue_type" form:"queue_type"`
}

type DishQueueParam struct {
	Type      string `json:"type" form:"type"`
	MemberId  string `json:"member_id" form:"member_id"`
	TaskID    uint   `json:"task_id" form:"task_id"`
	Result    string `json:"result" form:"result"`
	IsDie     bool   `json:"is_die" form:"is_die"`
	QueueType string `json:"queue_type" form:"queue_type"`
}

type ReportQueueParam struct {
	Type      string `json:"type" form:"type"`
	MemberId  string `json:"member_id" form:"member_id"`
	Week      string `json:"week" form:"week"`
	Year      string `json:"year" form:"year"`
	StartDate string `json:"start_date" form:"start_date"`
	EndDate   string `json:"end_date" form:"end_date"`
	TaskID    uint   `json:"task_id" form:"task_id"`
	Result    string `json:"result" form:"result"`
	IsDie     bool   `json:"is_die" form:"is_die"`
	QueueType string `json:"queue_type" form:"queue_type"`
}
