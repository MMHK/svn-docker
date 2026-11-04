// Package view contains HTTP DTOs only: JSON tags and Swagger examples.
// No business logic and no imports of the model or transport layers.
package view

// ErrorDetail is the machine readable part of an error response.
type ErrorDetail struct {
	Code    string            `json:"code" example:"validation_error"`
	Message string            `json:"message" example:"name must not be empty"`
	Details map[string]string `json:"details,omitempty"`
}

// ErrorEnvelope is the single error shape used by every endpoint.
type ErrorEnvelope struct {
	Error ErrorDetail `json:"error"`
}

// Health is returned by GET /healthz.
type Health struct {
	Status string `json:"status" example:"ok"`
}

// ListMeta is embedded by list responses.
type ListMeta struct {
	Page  int `json:"page" example:"1"`
	Size  int `json:"size" example:"50"`
	Total int `json:"total" example:"1"`
}
