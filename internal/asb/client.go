package asb

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/admin"
)

type AuthStatus struct {
	Ready   bool
	Message string
}

type QueueSnapshot struct {
	Name      string
	Active    int64
	Scheduled int64
	Dead      int64
	Transfer  int64
}

type Client struct {
	admin *admin.Client
	data  *azservicebus.Client
}

type DeadLetterMessage struct {
	MessageID                  string
	SequenceNumber             int64
	DeliveryCount              uint32
	DeadLetterReason           string
	DeadLetterErrorDescription string
	Body                       string
}

const (
	DLQFetchModePeek             = "peek"
	DLQFetchModePeekLock         = "peeklock"
	DLQFetchModeReceiveAndDelete = "receiveanddelete"
)

func NewClient(namespace string) (*Client, AuthStatus) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, AuthStatus{Ready: false, Message: fmt.Sprintf("Credential setup failed: %v", err)}
	}

	adminClient, err := admin.NewClient(namespace, cred, nil)
	if err != nil {
		return nil, AuthStatus{Ready: false, Message: fmt.Sprintf("Failed to create ASB admin client: %v", err)}
	}

	dataClient, err := azservicebus.NewClient(namespace, cred, nil)
	if err != nil {
		return nil, AuthStatus{Ready: false, Message: fmt.Sprintf("Failed to create ASB data client: %v", err)}
	}

	return &Client{admin: adminClient, data: dataClient}, AuthStatus{Ready: true, Message: "DefaultAzureCredential initialized"}
}

func (c *Client) ListQueues(ctx context.Context) ([]QueueSnapshot, error) {
	pager := c.admin.NewListQueuesRuntimePropertiesPager(nil)
	queues := make([]QueueSnapshot, 0)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, q := range page.QueueRuntimeProperties {
			queues = append(queues, QueueSnapshot{
				Name:      q.QueueName,
				Active:    int64(q.ActiveMessageCount),
				Scheduled: int64(q.ScheduledMessageCount),
				Dead:      int64(q.DeadLetterMessageCount),
				Transfer:  int64(q.TransferDeadLetterMessageCount),
			})
		}
	}

	return queues, nil
}

func (c *Client) GetQueue(ctx context.Context, queueName string) (QueueSnapshot, error) {
	resp, err := c.admin.GetQueueRuntimeProperties(ctx, queueName, nil)
	if err != nil {
		return QueueSnapshot{}, err
	}

	return QueueSnapshot{
		Name:      resp.QueueName,
		Active:    int64(resp.ActiveMessageCount),
		Scheduled: int64(resp.ScheduledMessageCount),
		Dead:      int64(resp.DeadLetterMessageCount),
		Transfer:  int64(resp.TransferDeadLetterMessageCount),
	}, nil
}

func (c *Client) FetchDeadLetterMessages(ctx context.Context, queueName string, mode string, maxMessages int) ([]DeadLetterMessage, error) {
	if c.data == nil {
		return nil, fmt.Errorf("data client unavailable")
	}
	if maxMessages <= 0 {
		return nil, fmt.Errorf("maxMessages must be greater than 0")
	}

	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	if normalizedMode == "" {
		normalizedMode = DLQFetchModePeek
	}

	var options azservicebus.ReceiverOptions
	options.SubQueue = azservicebus.SubQueueDeadLetter

	switch normalizedMode {
	case DLQFetchModePeek:
		options.ReceiveMode = azservicebus.ReceiveModePeekLock
	case DLQFetchModePeekLock:
		options.ReceiveMode = azservicebus.ReceiveModePeekLock
	case DLQFetchModeReceiveAndDelete:
		options.ReceiveMode = azservicebus.ReceiveModeReceiveAndDelete
	default:
		return nil, fmt.Errorf("unsupported fetch mode %q", mode)
	}

	receiver, err := c.data.NewReceiverForQueue(queueName, &options)
	if err != nil {
		return nil, err
	}
	defer receiver.Close(context.Background())

	var received []*azservicebus.ReceivedMessage

	switch normalizedMode {
	case DLQFetchModePeek:
		received, err = receiver.PeekMessages(ctx, maxMessages, nil)
		if err != nil {
			return nil, err
		}
	case DLQFetchModePeekLock, DLQFetchModeReceiveAndDelete:
		received, err = receiver.ReceiveMessages(ctx, maxMessages, nil)
		if err != nil {
			return nil, err
		}
	}

	messages := mapDeadLetterMessages(received)

	if normalizedMode == DLQFetchModePeekLock {
		for _, msg := range received {
			if err := receiver.AbandonMessage(ctx, msg, nil); err != nil {
				return nil, fmt.Errorf("failed to release message lock: %w", err)
			}
		}
	}

	return messages, nil
}

func mapDeadLetterMessages(received []*azservicebus.ReceivedMessage) []DeadLetterMessage {
	messages := make([]DeadLetterMessage, 0, len(received))
	for _, msg := range received {
		messages = append(messages, DeadLetterMessage{
			MessageID:                  msg.MessageID,
			SequenceNumber:             valueOrZero(msg.SequenceNumber),
			DeliveryCount:              msg.DeliveryCount,
			DeadLetterReason:           valueOrEmpty(msg.DeadLetterReason),
			DeadLetterErrorDescription: valueOrEmpty(msg.DeadLetterErrorDescription),
			Body:                       string(msg.Body),
		})
	}
	return messages
}

func valueOrZero(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
