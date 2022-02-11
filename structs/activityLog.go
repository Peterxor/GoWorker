package structs

type ActivityLogJsonModel struct {
	Type       string         `json:"type"`
	MemberID   string         `json:"member_id,omitempty"`
	MemberName string         `json:"member_name,omitempty"`
	Result     bool           `json:"result"`
	Statistic  StatisticModel `json:"statistic"`
	Message    string         `json:"message"`
	Messages   []ErrorModel   `json:"messages"`
}

type StatisticModel struct {
	TotalMember   int `json:"total_member"`
	ExpiredMember int `json:"expired_member"`
	FailMember    int `json:"fail_member"`
	OKMember      int `json:"ok_member"`
}
