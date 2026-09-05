package topology

import "testing"

func TestTopologyHasDurableEmailRouteForEachSubscriptionEvent(t *testing.T) {
	seen := map[RoutingKey]bool{}
	for _, router := range Topology {
		if !router.Exchange.Durable || !router.Queue.Durable {
			t.Fatalf("topology entry for %q must be durable", router.Binding.Route.RoutingKey)
		}
		if router.Queue.Name == QueueEmail {
			seen[router.Binding.Route.RoutingKey] = true
		}
	}
	for _, route := range []RoutingKey{RoutingSubscriptionConfirmation, RoutingSubscriptionCancellation} {
		if !seen[route] {
			t.Fatalf("email queue is missing route %q", route)
		}
	}
}

func TestRetryQueueReturnsMessagesToEventsExchange(t *testing.T) {
	if retryQueue.Args["x-message-ttl"] != RetryDelayMilliseconds {
		t.Fatalf("unexpected retry delay: %#v", retryQueue.Args["x-message-ttl"])
	}
	if retryQueue.Args["x-dead-letter-exchange"] != string(ExchangeEvents) {
		t.Fatalf("retry queue must return messages to events exchange: %#v", retryQueue.Args)
	}
}
