package asb

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
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
}

func NewClient(namespace string) (*Client, AuthStatus) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, AuthStatus{Ready: false, Message: fmt.Sprintf("Credential setup failed: %v", err)}
	}

	adminClient, err := admin.NewClient(namespace, cred, nil)
	if err != nil {
		return nil, AuthStatus{Ready: false, Message: fmt.Sprintf("Failed to create ASB admin client: %v", err)}
	}

	return &Client{admin: adminClient}, AuthStatus{Ready: true, Message: "DefaultAzureCredential initialized"}
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
