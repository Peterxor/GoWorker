package structs

type Report struct {
	//Mail           string   `json:"mail"`
	//ActivationCode string   `json:"activationCode"`
	Comment     string   `json:"comment"`
	CreatedBy   string   `json:"createdBy"`
	Calorie     JsonCalorie  `json:"calorie"`
	Water       JsonWater    `json:"water"`
	Days        []Day    `json:"days"`
	Nutrients   JsonNutrient `json:"nutrients"`
	Achievement string   `json:"achievement"`
	AchieveDetail AchieveDetail `json:"achieveDetail"`
}
