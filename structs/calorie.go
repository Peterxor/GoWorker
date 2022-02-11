package structs

type Calorie struct {
	Average float64 `json:"average"`
	Base    int     `json:"base"`
	Icon    int     `json:"icon"`
}

type JsonCalorie struct {
	Average int `json:"average"`
	Base    int `json:"base"`
	Icon    int `json:"icon"`
}
