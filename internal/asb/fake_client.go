package asb

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

const fakeLatency = 200 * time.Millisecond

type fakeQueueState struct {
	active    int64
	scheduled int64
	transfer  int64
	dlq       []DeadLetterMessage
}

type FakeClient struct {
	queues map[string]fakeQueueState
}

func NewFakeClient() (*FakeClient, AuthStatus) {
	return &FakeClient{
		queues: map[string]fakeQueueState{
			"billing-events": {
				active:    42,
				scheduled: 3,
				transfer:  0,
				dlq: []DeadLetterMessage{
					{
						MessageID:                  "billing-001",
						SequenceNumber:             1001,
						DeliveryCount:              6,
						DeadLetterReason:           "MaxDeliveryCountExceeded",
						DeadLetterErrorDescription: "Handler failed permanently",
						Body:                       `{"event":"invoice.created","invoiceId":"INV-1001","attempt":6}`,
					},
					{
						MessageID:                  "billing-002",
						SequenceNumber:             1002,
						DeliveryCount:              4,
						DeadLetterReason:           "DeserializationError",
						DeadLetterErrorDescription: "Unexpected field type",
						Body:                       `{"event":"invoice.updated","invoiceId":"INV-1002","amount":null}`,
					},
				},
			},
			"orders": {
				active:    12,
				scheduled: 1,
				transfer:  0,
				dlq: []DeadLetterMessage{
					{
						MessageID:                  "orders-001",
						SequenceNumber:             2001,
						DeliveryCount:              10,
						DeadLetterReason:           "MaxDeliveryCountExceeded",
						DeadLetterErrorDescription: "Downstream timeout",
						Body:                       `{"event":"order.submitted","orderId":"A-42","customer":"contoso"}`,
					},
				},
			},
			"shipments": {
				active:    8,
				scheduled: 0,
				transfer:  1,
				dlq:       []DeadLetterMessage{},
			},
		},
	}, AuthStatus{Ready: true, Message: "Fake client enabled (deterministic seed)"}
}

func (c *FakeClient) ListQueues(ctx context.Context) ([]QueueSnapshot, error) {
	if err := simulateLatencyWithContext(ctx); err != nil {
		return nil, err
	}

	names := c.sortedNames()
	queues := make([]QueueSnapshot, 0, len(names))
	for _, name := range names {
		state := c.queues[name]
		queues = append(queues, QueueSnapshot{
			Name:      name,
			Active:    state.active,
			Scheduled: state.scheduled,
			Dead:      int64(len(state.dlq)),
			Transfer:  state.transfer,
		})
	}
	return queues, nil
}

func (c *FakeClient) GetQueue(ctx context.Context, queueName string) (QueueSnapshot, error) {
	if err := simulateLatencyWithContext(ctx); err != nil {
		return QueueSnapshot{}, err
	}

	state, ok := c.queues[queueName]
	if !ok {
		return QueueSnapshot{}, fmt.Errorf("queue %q not found", queueName)
	}

	return QueueSnapshot{
		Name:      queueName,
		Active:    state.active,
		Scheduled: state.scheduled,
		Dead:      int64(len(state.dlq)),
		Transfer:  state.transfer,
	}, nil
}

func (c *FakeClient) FetchDeadLetterMessages(ctx context.Context, queueName string, mode string, maxMessages int) ([]DeadLetterMessage, error) {
	if err := simulateLatencyWithContext(ctx); err != nil {
		return nil, err
	}

	if maxMessages <= 0 {
		return nil, fmt.Errorf("maxMessages must be greater than 0")
	}

	state, ok := c.queues[queueName]
	if !ok {
		return nil, fmt.Errorf("queue %q not found", queueName)
	}

	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	if normalizedMode == "" {
		normalizedMode = DLQFetchModePeek
	}

	switch normalizedMode {
	case DLQFetchModePeek, DLQFetchModePeekLock, DLQFetchModeReceiveAndDelete:
	default:
		return nil, fmt.Errorf("unsupported fetch mode %q", mode)
	}

	count := maxMessages
	if len(state.dlq) < count {
		count = len(state.dlq)
	}

	selected := cloneMessages(state.dlq[:count])

	if normalizedMode == DLQFetchModeReceiveAndDelete {
		state.dlq = cloneMessages(state.dlq[count:])
		c.queues[queueName] = state
	}

	return selected, nil
}

func (c *FakeClient) sortedNames() []string {
	names := make([]string, 0, len(c.queues))
	for name := range c.queues {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func cloneMessages(messages []DeadLetterMessage) []DeadLetterMessage {
	cloned := make([]DeadLetterMessage, len(messages))
	copy(cloned, messages)
	return cloned
}

func simulateLatencyWithContext(ctx context.Context) error {
	if ctx == nil {
		time.Sleep(fakeLatency)
		return nil
	}

	select {
	case <-time.After(fakeLatency):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
