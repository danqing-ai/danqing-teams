package errs

import "errors"

var (
	ErrNotFound       = errors.New("not found")
	ErrInvalidInput   = errors.New("invalid input")
	ErrNotImplemented = errors.New("not implemented")
	ErrConflict       = errors.New("conflict")
)

// AppError carries an HTTP-friendly code.
type AppError struct {
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

func (e *AppError) Unwrap() error { return e.Err }

func NotFound(msg string) *AppError {
	return &AppError{Code: "not_found", Message: msg, Err: ErrNotFound}
}

func BadRequest(msg string) *AppError {
	return &AppError{Code: "bad_request", Message: msg, Err: ErrInvalidInput}
}
