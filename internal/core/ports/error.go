package ports

import (
	"errors"
	"fmt"
	"net/http"
)

type Code string
type Message string

const (
	CodeInternal         Code = "INTERNAL_ERROR"
	CodeBadRequest       Code = "INVALID_INPUT"
	CodeValidation       Code = "VALIDATION_ERROR"
	CodeNotFound         Code = "NOT_FOUND"
	CodeConflict         Code = "CONFLICT"
	CodeUnauthenticated  Code = "UNAUTHENTICATED"
	CodeUnauthorized     Code = "UNAUTHORIZED"
	CodeForbidden        Code = "FORBIDDEN"
	CodeMethodNotAllowed Code = "METHOD_NOT_ALLOWED"
	CodeTooManyRequests  Code = "TOO_MANY_REQUESTS"
	CodeUnavailable      Code = "SERVICE_UNAVAILABLE"
	CodeTimeout          Code = "TIMEOUT"
)

const (
	MessageInternal         = "an unexpected internal error occurred"
	MessageBadRequest       = "the request contains invalid input"
	MessageValidation       = "domain validation failed"
	MessageNotFound         = "the requested resource was not found"
	MessageConflict         = "the resource conflicts with the current state"
	MessageUnauthenticated  = "authentication is required"
	MessageUnauthorized     = "without authorization"
	MessageForbidden        = "the operation is not permitted"
	MessageMethodNotAllowed = "the HTTP method is not allowed"
	MessageTooManyRequests  = "too many requests"
	MessageUnavailable      = "the service is temporarily unavailable"
	MessageTimeout          = "the operation timed out"
)

type Details struct {
	Message Message
	Status  int
}

var definitions = map[Code]Details{
	CodeInternal:         {Message: MessageInternal, Status: http.StatusInternalServerError},
	CodeBadRequest:       {Message: MessageBadRequest, Status: http.StatusBadRequest},
	CodeValidation:       {Message: MessageValidation, Status: http.StatusUnprocessableEntity},
	CodeNotFound:         {Message: MessageNotFound, Status: http.StatusNotFound},
	CodeConflict:         {Message: MessageConflict, Status: http.StatusConflict},
	CodeUnauthenticated:  {Message: MessageUnauthenticated, Status: http.StatusUnauthorized},
	CodeUnauthorized:     {Message: MessageUnauthorized, Status: http.StatusUnauthorized},
	CodeForbidden:        {Message: MessageForbidden, Status: http.StatusForbidden},
	CodeMethodNotAllowed: {Message: MessageMethodNotAllowed, Status: http.StatusMethodNotAllowed},
	CodeTooManyRequests:  {Message: MessageTooManyRequests, Status: http.StatusTooManyRequests},
	CodeUnavailable:      {Message: MessageUnavailable, Status: http.StatusServiceUnavailable},
	CodeTimeout:          {Message: MessageTimeout, Status: http.StatusGatewayTimeout},
}

type Error struct {
	Code    Code
	Message Message
	Status  int
	Cause   error
}

func (err *Error) Error() string {
	if err == nil {
		return "<nil>"
	}

	if err.Cause != nil {
		return fmt.Sprintf("[%s:%d] %s: %v", err.Code, err.Status, err.Message, err.Cause)
	}

	return fmt.Sprintf("[%s:%d] %s", err.Code, err.Status, err.Message)
}

func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}

	return err.Cause
}

func New(code Code, message Message, status int, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Status:  status,
		Cause:   cause,
	}
}

func Wrap(code Code, cause error) error {
	if cause == nil {
		return nil
	}

	return kind(code, cause)
}

func BadRequest(cause error) *Error {
	return kind(CodeBadRequest, cause)
}

func NotFound(cause error) *Error {
	return kind(CodeNotFound, cause)
}

func Conflict(cause error) *Error {
	return kind(CodeConflict, cause)
}

func Unauthenticated(cause error) *Error {
	return kind(CodeUnauthenticated, cause)
}

func Forbidden(cause error) *Error {
	return kind(CodeForbidden, cause)
}

func MethodNotAllowed(cause error) *Error {
	return kind(CodeMethodNotAllowed, cause)
}

func TooManyRequests(cause error) *Error {
	return kind(CodeTooManyRequests, cause)
}

func Unavailable(cause error) *Error {
	return kind(CodeUnavailable, cause)
}

func Unauthorized(cause error) *Error {
	return kind(CodeUnauthorized, cause)
}

func Timeout(cause error) error {
	return Wrap(CodeTimeout, cause)
}

func Internal(cause error) error {
	return Wrap(CodeInternal, cause)
}

func As(err error) (*Error, bool) {
	var porterror *Error
	if !errors.As(err, &porterror) {
		return nil, false
	}

	return porterror, true
}

func IsCode(err error, code Code) bool {
	e, ok := As(err)
	return ok && e.Code == code
}

func kind(code Code, cause error) *Error {
	details, ok := definitions[code]
	if !ok {
		return &Error{
			Code:    code,
			Message: MessageInternal,
			Status:  http.StatusInternalServerError,
			Cause:   cause,
		}
	}

	return &Error{
		Code:    code,
		Message: details.Message,
		Status:  details.Status,
		Cause:   cause,
	}
}
