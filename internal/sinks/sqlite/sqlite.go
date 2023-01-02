package sqliteSink

import (
	"context"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"

	_ "embed"

	v1 "k8s.io/api/core/v1"
)

var (
	//go:embed table.sql
	createTableQuery string
	//go:embed insert.sql
	insertRowQuery string
)

type SqliteSink struct {
	db *sql.DB
}

func New(dbPath string) (*SqliteSink, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(createTableQuery); err != nil {
		return nil, err
	}

	return &SqliteSink{
		db: db,
	}, nil
}

func (s *SqliteSink) Write(ctx context.Context, event v1.Event) error {
	statement, err := s.db.Prepare(insertRowQuery)
	if err != nil {
		return err
	}
	_, err = statement.Exec(
		event.UID,
		event.Type,
		event.Reason,
		event.Message,
		event.Action,
		event.Count,
		event.ReportingController,
		event.ReportingInstance,
		event.FirstTimestamp.String(),
		event.LastTimestamp.String(),
		event.InvolvedObject.UID,
		event.InvolvedObject.APIVersion,
		event.InvolvedObject.Kind,
		event.InvolvedObject.Name,
		event.InvolvedObject.Namespace,
	)

	return err
}

func (s *SqliteSink) Query(ctx context.Context, query string) ([]map[string]interface{}, error) {
	return nil, nil
}

func (s *SqliteSink) Start(_ context.Context) error {
	return nil
}

func (s *SqliteSink) Stop() error {
	return s.db.Close()
}
