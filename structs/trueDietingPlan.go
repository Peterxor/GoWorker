package structs

type TrueDietingPlan struct {
	Fats              float64    `json:"fats"`
	CrudeFat          float64    `json:"crude_fat"`
	Water             float64    `json:"water"`
	Protein           float64    `json:"protein"`
	CrudeProtein      float64    `json:"crude_protein"`
	Calories          float64    `json:"calories"`
	FixedKcal         float64    `json:"fixed_kcal"`
	UpdatedAt         string `json:"updated_at"`
	Carbohydrate      float64    `json:"carbohydrate"`
	TotalCarbohydrate float64    `json:"total_carbohydrate"`
	Dietaryfiber      float64    `json:"dietaryFiber"`
	DietaryfiberDash  float64    `json:"dietary_fiber"`
}
