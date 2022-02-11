package structs

type Nutrient struct {
	Protein       NutrientItem `json:"protein"`
	Fat           NutrientItem `json:"fat"`
	Carbohydrates NutrientItem `json:"carbohydrates"`
	Fibre         NutrientItem `json:"fibre"`
}

type NutrientItem struct {
	Total   float64 `json:"total"`
	Suggest float64 `json:"suggest"`
}

type JsonNutrient struct {
	Protein       JsonNutrientItem `json:"protein"`
	Fat           JsonNutrientItem `json:"fat"`
	Carbohydrates JsonNutrientItem `json:"carbohydrates"`
	Fibre         JsonNutrientItem `json:"fibre"`
}

type JsonNutrientItem struct {
	Total   int `json:"total"`
	Suggest int `json:"suggest"`
}
