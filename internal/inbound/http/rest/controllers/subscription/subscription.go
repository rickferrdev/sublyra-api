package subscription

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rickferrdev/sublyra-api/internal/core/ports"
	"github.com/rickferrdev/sublyra-api/internal/core/services/subscription"
	"go.uber.org/fx"
)

var Invoke = fx.Invoke(New)

type Controller struct {
	router  fiber.Router
	service subscription.Interface
}

type FxParams struct {
	fx.In

	FiberRouter fiber.Router
	Service     subscription.Interface
}

func New(params FxParams) *Controller {
	controller := &Controller{
		router:  params.FiberRouter,
		service: params.Service,
	}

	controller.router.Post("/subscription", controller.Subscription)
	controller.router.Post("/subscription/confirm", controller.SubscriptionConfirm)
	controller.router.Post("/unsubscription", controller.Unsubscription)
	controller.router.Post("/unsubscription/confirm", controller.UnsubscriptionConfirm)

	return controller
}

func (controller *Controller) Subscription(c fiber.Ctx) error {
	var body RequestSubscriptionDTO
	if err := c.Bind().JSON(&body); err != nil {
		return ports.BadRequest(err)
	}

	if err := controller.service.RegisterSubscription(c.Context(), body.Email); err != nil {
		return err
	}

	return c.Status(fiber.StatusAccepted).JSON(ResponseDTO{
		Code:    CodeSubscriptionPending,
		Message: MessageSubscriptionPending,
	})
}

func (controller *Controller) SubscriptionConfirm(c fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return ports.BadRequest(nil)
	}

	if err := controller.service.SubscriptionConfirm(c.Context(), token); err != nil {
		return err
	}

	return c.Status(fiber.StatusAccepted).JSON(ResponseDTO{
		Code:    CodeSubscriptionConfirmed,
		Message: MessageSubscriptionConfirmed,
	})
}

func (controller *Controller) Unsubscription(c fiber.Ctx) error {
	var body RequestUnsubscriptionDTO
	if err := c.Bind().JSON(&body); err != nil {
		return ports.BadRequest(err)
	}

	if err := controller.service.RegisterUnsubscription(c.Context(), body.Email); err != nil {
		return err
	}

	return c.Status(fiber.StatusAccepted).JSON(ResponseDTO{
		Code:    CodeUnsubscriptionPending,
		Message: MessageUnsubscriptionPending,
	})
}

func (controller *Controller) UnsubscriptionConfirm(c fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return ports.BadRequest(nil)
	}

	if err := controller.service.UnsubscriptionConfirm(c.Context(), token); err != nil {
		return err
	}

	return c.Status(fiber.StatusAccepted).JSON(ResponseDTO{
		Code:    CodeUnsubscriptionConfirmed,
		Message: MessageUnsubscriptionConfirmed,
	})
}
