package httputils

import "fmt"

type HTTPErrorResponse struct {
	StatusCode   int    `json:"-"`
	ErrorMessage string `json:"description,omitempty"`
	ErrorKey     string `json:"error,omitempty"`
}

func (e HTTPErrorResponse) Error() string {
	return fmt.Sprintf("Status: %v; ErrorKey: %v; ErrorMessage: %v", e.StatusCode, e.ErrorKey, e.ErrorMessage)
}
