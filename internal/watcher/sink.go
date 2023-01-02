package watcher

import (
	"context"

	v1 "k8s.io/api/core/v1"
)

type Sink interface {
	Write(ctx context.Context, event v1.Event) error
	Start(ctx context.Context) error
	Stop() error
}
