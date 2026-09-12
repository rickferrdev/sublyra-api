package health

type ResponseHealthDTO struct {
	Status   string                `json:"status"`
	Services map[string]ServiceDTO `json:"services"`
}

type ServiceDTO struct {
	Status string `json:"status"`
}
