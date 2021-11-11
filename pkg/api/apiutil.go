package api

type GenericErrorResponse struct {
	ID      string   `json:"id,omitempty"`
	Message string   `json:"message,omitempty"`
	Errors  []Errors `json:"errors,omitempty"`
}
type Errors struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}
