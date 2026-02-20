package asb

import (
	"context"
	"testing"
)

func TestNewFakeClientReturnsReadyAuthStatus(t *testing.T) {
	client, auth := NewFakeClient()
	if client == nil {
		t.Fatal("expected fake client")
	}
	if !auth.Ready {
		t.Fatalf("expected ready auth status, got %#v", auth)
	}
}

func TestFakeClientListQueuesIsDeterministic(t *testing.T) {
	client, _ := NewFakeClient()
	queues, err := client.ListQueues(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(queues) != 3 {
		t.Fatalf("expected 3 queues, got %d", len(queues))
	}
	if queues[0].Name != "billing-events" || queues[1].Name != "orders" || queues[2].Name != "shipments" {
		t.Fatalf("unexpected queue order: %#v", queues)
	}
}

func TestFakeClientPeekDoesNotRemoveMessages(t *testing.T) {
	client, _ := NewFakeClient()

	messages, err := client.FetchDeadLetterMessages(context.Background(), "billing-events", DLQFetchModePeek, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	queue, err := client.GetQueue(context.Background(), "billing-events")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if queue.Dead != 2 {
		t.Fatalf("expected dead-letter count to remain 2, got %d", queue.Dead)
	}
}

func TestFakeClientReceiveAndDeleteRemovesMessages(t *testing.T) {
	client, _ := NewFakeClient()

	messages, err := client.FetchDeadLetterMessages(context.Background(), "billing-events", DLQFetchModeReceiveAndDelete, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	queue, err := client.GetQueue(context.Background(), "billing-events")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if queue.Dead != 1 {
		t.Fatalf("expected dead-letter count 1 after deletion, got %d", queue.Dead)
	}
}
