//go:build integration

package rabbitmq_test

import (
	"context"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	infrarabbit "github.com/rickferrdev/sublyra-api/internal/infra/rabbitmq"
	"github.com/rickferrdev/sublyra-api/internal/outbound/rabbitmq/publisher"
	"github.com/rickferrdev/sublyra-api/internal/outbound/rabbitmq/topology"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

func TestRabbitMQ_Integration(t *testing.T) {
	ctx := context.Background()

	rabbitmqContainer, err := rabbitmq.Run(ctx, "rabbitmq:4-management")
	if err != nil {
		t.Fatalf("failed to start rabbitmq container: %s", err)
	}
	t.Cleanup(func() {
		if err := rabbitmqContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	amqpURI, err := rabbitmqContainer.AmqpURL(ctx)
	if err != nil {
		t.Fatalf("failed to get amqp url: %s", err)
	}

	conn, err := amqp.Dial(amqpURI)
	if err != nil {
		t.Fatalf("failed to connect to rabbitmq: %s", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	channel, err := conn.Channel()
	if err != nil {
		t.Fatalf("failed to open channel: %s", err)
	}
	t.Cleanup(func() {
		_ = channel.Close()
	})

	rabbitClient := &infrarabbit.Client{
		Connection: conn,
		Channel:    channel,
	}

	// Declare topology
	if err := topology.Declare(rabbitClient); err != nil {
		t.Fatalf("failed to declare rabbitmq topology: %v", err)
	}

	// Create Publisher
	pub := publisher.New(publisher.FxParams{
		Client: rabbitClient,
	})

	// Test publishing a message to subscription confirmation route
	testPayload := []byte(`{"event_id":"evt_123","event_type":"subscription_confirmation_requested","payload":{"email":"test@example.com"}}`)

	err = pub.Publish(ctx, publisher.Message{
		Route:   topology.RouteSubscriptionConfirmation,
		Payload: testPayload,
	})
	if err != nil {
		t.Fatalf("expected message publishing success, got: %v", err)
	}

	// Consume message from sublyra.email queue to verify delivery
	msgs, err := channel.Consume(
		string(topology.QueueEmail),
		"test-consumer",
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		t.Fatalf("failed to consume from queue: %v", err)
	}

	select {
	case msg := <-msgs:
		if string(msg.Body) != string(testPayload) {
			t.Errorf("expected body %s, got %s", string(testPayload), string(msg.Body))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for message from queue sublyra.email")
	}
}
