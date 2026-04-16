package apperr

import "net/http"

const (
	CodeOK              = 0
	CodeInvalidArgument = 10001
	CodeUnauthorized    = 10002
	CodeForbidden       = 10003
	CodeNotFound        = 10004
	CodeInternal        = 10005
)

type AppError struct {
	Code       int
	Message    string
	HTTPStatus int
}

func (e *AppError) Error() string {
	return e.Message
}

func New(code int, message string, status int) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: status}
}

func InvalidArgument(message string) *AppError {
	return New(CodeInvalidArgument, message, http.StatusBadRequest)
}

func NotFound(message string) *AppError {
	return New(CodeNotFound, message, http.StatusNotFound)
}

func Unauthorized(message string) *AppError {
	return New(CodeUnauthorized, message, http.StatusUnauthorized)
}

func Internal(message string) *AppError {
	return New(CodeInternal, message, http.StatusInternalServerError)
}
