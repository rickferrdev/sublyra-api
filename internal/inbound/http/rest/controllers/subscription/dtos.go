package subscription

const (
	CodeSubscriptionPending   ResponseCode = "SUBSCRIPTION_PENDING"
	CodeSubscriptionConfirmed ResponseCode = "SUBSCRIPTION_CONFIRMED"

	CodeUnsubscriptionPending   ResponseCode = "UNSUBSCRIPTION_PENDING"
	CodeUnsubscriptionConfirmed ResponseCode = "UNSUBSCRIPTION_CONFIRMED"

	MessageSubscriptionPending   ResponseMessage = "Subscription in pending confirmation status"
	MessageSubscriptionConfirmed ResponseMessage = "Subscription successfully confirmed"

	MessageUnsubscriptionPending   ResponseMessage = "Subscription in pending unsubscription status"
	MessageUnsubscriptionConfirmed ResponseMessage = "Subscription cancelled confirmed"
)

type (
	ResponseCode    string
	ResponseMessage string

	ResponseDTO struct {
		Code    ResponseCode    `json:"code"`
		Message ResponseMessage `json:"message"`
	}

	RequestSubscriptionDTO struct {
		Email string `json:"email" validate:"required,email"`
	}

	RequestUnsubscriptionDTO struct {
		Email string `json:"email" validate:"required,email"`
	}
)
