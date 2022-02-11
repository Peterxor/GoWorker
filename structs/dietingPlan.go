package structs

type DietingPlan struct {
	Fats         float64   `json:"fats"`
	Water        float64    `json:"water"`
	Protein      float64    `json:"protein"`
	Calories     float64    `json:"calories"`
	UpdatedAt    string `json:"updated_at"`
	Carbohydrate float64    `json:"carbohydrate"`
	Dietaryfiber float64    `json:"dietaryFiber"`
}
