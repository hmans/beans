package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hmans/beans/internal/bean"
)

// Error codes for JSON responses
const (
	ErrNotFound      = "NOT_FOUND"
	ErrNoBeansDir    = "NO_BEANS_DIR"
	ErrInvalidStatus = "INVALID_STATUS"
	ErrFileError     = "FILE_ERROR"
	ErrValidation    = "VALIDATION_ERROR"
)

// Response is the standard JSON response envelope.
type Response struct {
	Success  bool         `json:"success"`
	Bean     *bean.Bean   `json:"bean,omitempty"`
	Beans    []*bean.Bean `json:"beans,omitempty"`
	Count    int          `json:"count,omitempty"`
	Message  string       `json:"message,omitempty"`
	Warnings []string     `json:"warnings,omitempty"`
	Error    string       `json:"error,omitempty"`
	Code     string       `json:"code,omitempty"`
	Path     string       `json:"path,omitempty"`
}

// JSON outputs a response as JSON to stdout.
func JSON(resp Response) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

// Success outputs a successful single-bean response.
func Success(b *bean.Bean, message string) error {
	return JSON(Response{
		Success: true,
		Bean:    b,
		Message: message,
	})
}

// SuccessWithWarnings outputs a successful single-bean response with warnings.
func SuccessWithWarnings(b *bean.Bean, message string, warnings []string) error {
	return JSON(Response{
		Success:  true,
		Bean:     b,
		Message:  message,
		Warnings: warnings,
	})
}

// SuccessSingle outputs a single bean directly (no wrapper).
// This allows intuitive jq usage: beans show --json <id> | jq '.title'
func SuccessSingle(b *bean.Bean) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(b)
}

// SuccessMultiple outputs a bean array directly (no wrapper).
// This allows intuitive jq usage: beans list --json | jq '.[]'
func SuccessMultiple(beans []*bean.Bean) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(beans)
}

// SuccessMessage outputs a success response with just a message.
func SuccessMessage(message string) error {
	return JSON(Response{
		Success: true,
		Message: message,
	})
}

// SuccessInit outputs a success response for init command.
func SuccessInit(path string) error {
	return JSON(Response{
		Success: true,
		Message: "Initialized .beans directory",
		Path:    path,
	})
}

// Error outputs an error response and returns an error for command handling.
func Error(code string, message string) error {
	_ = JSON(Response{
		Success: false,
		Error:   message,
		Code:    code,
	})
	return fmt.Errorf("%s", message)
}

// ErrorFrom outputs an error response from an existing error.
func ErrorFrom(code string, err error) error {
	return Error(code, err.Error())
}
