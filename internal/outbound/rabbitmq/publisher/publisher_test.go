package publisher

import (
	"context"
	"testing"

	"github.com/rickferrdev/sublyra-api/internal/core/ports"
)

func TestPublishRejectsInvalidJSONBeforeUsingChannel(t *testing.T) {
	publisher := &Publisher{}
	err := publisher.Publish(context.Background(), Message{Payload: []byte("not-json")})
	if !ports.IsCode(err, "RABBITMQ_PUBLISH_ERROR") {
		t.Fatalf("expected publish error for invalid JSON, got %v", err)
	}
}
