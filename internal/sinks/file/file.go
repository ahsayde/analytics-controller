package fileSink

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	v1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type FilesystemSink struct {
	file      *os.File
	eventChan chan v1.Event
}

func New(filePath string) (*FilesystemSink, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s : %w", filePath, err)
	}
	return &FilesystemSink{
		file:      file,
		eventChan: make(chan v1.Event, 50),
	}, nil
}

func (s *FilesystemSink) Write(ctx context.Context, event v1.Event) error {
	s.eventChan <- event
	return nil
}

func (f *FilesystemSink) worker(ctx context.Context) {
	for {
		select {
		case event := <-f.eventChan:
			if err := json.NewEncoder(f.file).Encode(event); err != nil {
				log.Log.Error(err, "failed to write event to file")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (f *FilesystemSink) Start(ctx context.Context) error {
	go f.worker(ctx)
	return nil
}

func (f *FilesystemSink) Stop() error {
	close(f.eventChan)
	defer f.file.Close()
	if err := f.file.Sync(); err != nil {
		return fmt.Errorf("failed to write all results to file : %w", err)
	}
	return nil
}
