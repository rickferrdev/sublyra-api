package rabbitmq

import (
	"errors"
	"net/http"

	"github.com/rickferrdev/sublyra-api/internal/core/ports"
)

const (
	CodeRabbitMQConnection ports.Code = "RABBITMQ_CONNECTION_ERROR"
	CodeRabbitMQPublish    ports.Code = "RABBITMQ_PUBLISH_ERROR"
	CodeRabbitMQConsume    ports.Code = "RABBITMQ_CONSUME_ERROR"
	CodeRabbitMQAck        ports.Code = "RABBITMQ_ACK_ERROR"
	CodeRabbitMQNack       ports.Code = "RABBITMQ_NACK_ERROR"

	CodeWorkerPayloadInvalid  ports.Code = "CODE_WORKER_PAYLOAD_INVALID"
	CodeLimitAttemptsExceeded ports.Code = "CODE_LIMIT_ATTEMPTS_EXCEEDED"
)

const (
	MessageRabbitMQConnection ports.Message = "rabbitmq connection error"
	MessageRabbitMQPublish    ports.Message = "rabbitmq publish error"
	MessageRabbitMQConsume    ports.Message = "rabbitmq consume error"
	MessageRabbitMQAck        ports.Message = "rabbitmq ack error"
	MessageRabbitMQNack       ports.Message = "rabbitmq nack error"

	MessageWorkerPayloadInvalid  ports.Message = "payload processed by the worker is invalid"
	MessageLimitAttemptsExceeded ports.Message = "limit of attempts exceeded"
	MessageDuplicateExists       ports.Message = "subscription duplicate"
	MessageNotAcknowledged       ports.Message = "was not acknowledged"
	MessageUpdateFailed          ports.Message = "failed to update subscription status"
	MessageNotFound              ports.Message = "subscription was not found"
	MessageObjectIDInvalid       ports.Message = "did not expect to receive an invalid or empty object id"
	MessageFailedStartSession    ports.Message = "failed to start a database session"
)

var (
	ErrRabbitMQConnection = errors.New(string(MessageRabbitMQConnection))
	ErrRabbitMQPublish    = errors.New(string(MessageRabbitMQPublish))
	ErrRabbitMQConsume    = errors.New(string(MessageRabbitMQConsume))
	ErrRabbitMQAck        = errors.New(string(MessageRabbitMQAck))
	ErrRabbitMQNack       = errors.New(string(MessageRabbitMQNack))

	ErrWorkerPayloadInvalid  = errors.New(string(MessageWorkerPayloadInvalid))
	ErrLimitAttemptsExceeded = errors.New(string(MessageLimitAttemptsExceeded))
)

func InternalError(cause error) *ports.Error {
	return ports.New(ports.CodeInternal, ports.MessageInternal, http.StatusInternalServerError, cause)
}

func RabbitMQConnectionError(cause error) *ports.Error {
	return ports.New(CodeRabbitMQConnection, MessageRabbitMQConnection, http.StatusInternalServerError, cause)
}

func RabbitMQPublishError(cause error) *ports.Error {
	return ports.New(CodeRabbitMQPublish, MessageRabbitMQPublish, http.StatusInternalServerError, cause)
}

func RabbitMQConsumeError(cause error) *ports.Error {
	return ports.New(CodeRabbitMQConsume, MessageRabbitMQConsume, http.StatusInternalServerError, cause)
}

func RabbitMQAckError(cause error) *ports.Error {
	return ports.New(CodeRabbitMQAck, MessageRabbitMQAck, http.StatusInternalServerError, cause)
}

func RabbitMQNackError(cause error) *ports.Error {
	return ports.New(CodeRabbitMQNack, MessageRabbitMQNack, http.StatusInternalServerError, cause)
}

func WorkerPayloadInvalidError(cause error) *ports.Error {
	return ports.New(CodeWorkerPayloadInvalid, MessageWorkerPayloadInvalid, http.StatusBadRequest, cause)
}

func LimitAttemptsExceededError(cause error) *ports.Error {
	return ports.New(CodeLimitAttemptsExceeded, MessageLimitAttemptsExceeded, http.StatusTooManyRequests, cause)
}
