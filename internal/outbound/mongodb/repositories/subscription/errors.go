package subscription

import (
	"errors"
	"net/http"

	"github.com/rickferrdev/sublyra-api/internal/core/ports"
)

const (
	CodeDuplicateExists    ports.Code = "SUBSCRIPTION_DATABASE_DUPLICATE_EXISTS"
	CodeNotAcknowledged    ports.Code = "SUBSCRIPTION_DATABASE_NOT_ACKNOWLEDGED"
	CodeUpdateFailed       ports.Code = "SUBSCRIPTION_DATABASE_UPDATE_FAILED"
	CodeNotFound           ports.Code = "SUBSCRIPTION_DATABASE_NOT_FOUND"
	ObjectIDInvalid        ports.Code = "SUBSCRIPTION_DATABASE_OBJECT_ID_INVALID"
	CodeFailedStartSession ports.Code = "SUBSCRIPTION_DATABASE_FAILED_START_SESSION"
)

const (
	MessageDuplicateExists    ports.Message = "subscription duplicate"
	MessageNotAcknowledged    ports.Message = "was not acknowledged"
	MessageUpdateFailed       ports.Message = "failed to update subscription status"
	MessageNotFound           ports.Message = "subscription was not found"
	MessageObjectIDInvalid    ports.Message = "did not expect to receive an invalid or empty object id"
	MessageFailedStartSession ports.Message = "failed to start a database session"
)

var (
	ErrDuplicateExists    = errors.New(string(MessageDuplicateExists))
	ErrNotAcknowledged    = errors.New(string(MessageNotAcknowledged))
	ErrUpdateFailed       = errors.New(string(MessageUpdateFailed))
	ErrNotFound           = errors.New(string(MessageNotFound))
	ErrUUIDInvalid        = errors.New(string(MessageObjectIDInvalid))
	ErrFailedStartSession = errors.New(string(MessageFailedStartSession))
)

func InternalError(cause error) *ports.Error {
	return ports.New(ports.CodeInternal, ports.MessageInternal, http.StatusInternalServerError, cause)
}

func DuplicateExistsError(cause error) *ports.Error {
	return ports.New(CodeDuplicateExists, MessageDuplicateExists, http.StatusConflict, cause)
}

func NotAcknowledgedError(cause error) *ports.Error {
	return ports.New(CodeNotAcknowledged, MessageNotAcknowledged, http.StatusInternalServerError, cause)
}

func UpdateFailedError(cause error) *ports.Error {
	return ports.New(CodeUpdateFailed, MessageUpdateFailed, http.StatusInternalServerError, cause)
}

func NotFoundError(cause error) *ports.Error {
	return ports.New(CodeNotFound, MessageNotFound, http.StatusNotFound, cause)
}

func ObjectIDInvalidError(cause error) *ports.Error {
	return ports.New(ObjectIDInvalid, MessageObjectIDInvalid, http.StatusInternalServerError, cause)
}

func FailedStartSessionError(cause error) *ports.Error {
	return ports.New(CodeFailedStartSession, MessageFailedStartSession, http.StatusInternalServerError, cause)
}
