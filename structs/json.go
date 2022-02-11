package structs

type JsonModel struct {
	Data  []int64  `json:"data" form:"data"`
	Other []string `json:"other" form:"other"`
	Type  string   `json:"type" form:"type"`
}
