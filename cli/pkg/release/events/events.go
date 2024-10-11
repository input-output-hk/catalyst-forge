package events

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/lib/project/project"
)

type EventType string

const (
	MergeEventName EventType = "merge"
	TagEventName   EventType = "tag"
)

type ReleaseEvent interface {
	Firing() (bool, error)
}

// Fires returns true if any of the given release events are firing.
func Firing(project *project.Project, events []string, logger *slog.Logger) bool {
	for _, event := range events {
		logger.Debug("checking release event", "event", event)
		releaseEvent, err := getReleaseEvent(project, EventType(event), logger)
		if err != nil {
			logger.Error("failed to get release event", "error", err)
			continue
		}

		firing, err := releaseEvent.Firing()
		if err != nil {
			logger.Error("failed to check if release event is firing", "error", err)
			continue
		}

		if firing {
			logger.Debug("release event is firing", "event", event)
			return true
		}
	}

	return false
}

// GetReleaseEvent returns a release event based on the given event type.
func getReleaseEvent(project *project.Project, eventType EventType, logger *slog.Logger) (ReleaseEvent, error) {
	switch eventType {
	case MergeEventName:
		return &MergeEvent{
			logger:  logger,
			project: project,
		}, nil
	case TagEventName:
		return &TagEvent{
			logger:  logger,
			project: project,
		}, nil
	default:
		return nil, fmt.Errorf("unknown release event type: %s", eventType)
	}
}
