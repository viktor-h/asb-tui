package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/viktor/asb-tui/internal/asb"
	"github.com/viktor/asb-tui/internal/config"
	"github.com/viktor/asb-tui/internal/ui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	asbClient, authStatus := asb.NewClient(cfg.Namespace)

	var fetcher func(ctx context.Context) ([]ui.QueueMetrics, error)
	var fetchOne func(ctx context.Context, queueName string) (ui.QueueMetrics, error)
	if asbClient != nil {
		fetcher = func(ctx context.Context) ([]ui.QueueMetrics, error) {
			snapshots, err := asbClient.ListQueues(ctx)
			if err != nil {
				return nil, err
			}
			return toQueueMetrics(snapshots), nil
		}

		fetchOne = func(ctx context.Context, queueName string) (ui.QueueMetrics, error) {
			snapshot, err := asbClient.GetQueue(ctx, queueName)
			if err != nil {
				return ui.QueueMetrics{}, err
			}
			return ui.QueueMetrics{
				Name:      snapshot.Name,
				Active:    snapshot.Active,
				Scheduled: snapshot.Scheduled,
				Dead:      snapshot.Dead,
				Transfer:  snapshot.Transfer,
			}, nil
		}
	}

	program := tea.NewProgram(ui.NewModel(cfg, authStatus, fetcher, fetchOne), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "runtime error: %v\n", err)
		os.Exit(1)
	}
}

func toQueueMetrics(snapshots []asb.QueueSnapshot) []ui.QueueMetrics {
	queues := make([]ui.QueueMetrics, 0, len(snapshots))
	for _, q := range snapshots {
		queues = append(queues, ui.QueueMetrics{
			Name:      q.Name,
			Active:    q.Active,
			Scheduled: q.Scheduled,
			Dead:      q.Dead,
			Transfer:  q.Transfer,
		})
	}
	return queues
}
