package structs

type Water struct {
	Average float64 `json:"average"`
	Base    int     `json:"base"`
	Icon    int     `json:"icon"`
}

type JsonWater struct {
	Average int `json:"average"`
	Base    int `json:"base"`
	Icon    int `json:"icon"`
}
