package elasticSink

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"io"
	"io/ioutil"
	"time"

	elastic "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	resultChanSize  int           = 50
	retriesInterval time.Duration = 500 * time.Millisecond
	retries         int           = 5
)

//go:embed schema.json
var schema []byte

type IndexTemplate struct {
	Index Index `json:"index"`
}

type Index struct {
	IndexName string `json:"_index"`
	Id        string `json:"_id"`
}

type ElasticSink struct {
	client      *elastic.Client
	indexName   string
	eventBatch  []v1.Event
	eventChan   chan v1.Event
	batchExpiry time.Duration
}

// New returns a sink that sends results to elasticsearch index
func New(address, index, username, password string, batchSize, batchExpriry int) (*ElasticSink, error) {
	client, err := elastic.NewClient(
		elastic.Config{
			Addresses: []string{address},
			Username:  username,
			Password:  password,
		},
	)
	if err != nil {
		return nil, err
	}

	return &ElasticSink{
		eventChan:   make(chan v1.Event, resultChanSize),
		eventBatch:  make([]v1.Event, 0, batchSize),
		client:      client,
		indexName:   index,
		batchExpiry: time.Duration(batchExpriry * int(time.Second)),
	}, nil
}

func (es *ElasticSink) Write(_ context.Context, event v1.Event) error {
	es.eventChan <- event
	return nil
}

// Start starts the sink to send events when batch size is met or an interval has passed
func (es *ElasticSink) Start(ctx context.Context) error {
	go es.worker(ctx)
	return nil
}

func (es *ElasticSink) Stop() error {
	close(es.eventChan)
	return nil
}

func (es *ElasticSink) worker(ctx context.Context) error {
	timer := time.NewTicker(es.batchExpiry)
	for {
		select {
		case result := <-es.eventChan:
			es.eventBatch = append(es.eventBatch, result)
			if len(es.eventBatch) == cap(es.eventBatch) {
				es.writeBatch(ctx, es.eventBatch)
				es.eventBatch = es.eventBatch[:0]
				timer.Reset(es.batchExpiry)
			}
		case <-timer.C:
			if len(es.eventBatch) > 0 {
				es.writeBatch(ctx, es.eventBatch)
				es.eventBatch = es.eventBatch[:0]
			}
		case <-ctx.Done():
			if len(es.eventBatch) > 0 {
				es.writeBatch(ctx, es.eventBatch)
			}
			return ctx.Err()
		}
	}
}

func (es *ElasticSink) writeBatch(ctx context.Context, events []v1.Event) {
	body, err := es.createBody(events)
	if err != nil {
		log.Log.Error(err, "")
		return
	}

	req := esapi.BulkRequest{
		Body:  &body,
		Index: es.indexName,
	}

	resp, err := req.Do(ctx, es.client)
	if err != nil {
		log.Log.Error(err, "")
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode > 399 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Log.Error(err, "")
		}
		log.Log.Error(nil, "failed to write events", "body", string(body))
	}
}

func (es *ElasticSink) createBody(events []v1.Event) (bytes.Buffer, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	for _, event := range events {
		index := IndexTemplate{Index: Index{IndexName: es.indexName, Id: string(event.UID)}}
		if err := encoder.Encode(index); err != nil {
			if err != io.EOF {
				log.Log.Error(err, "")
			}
		}
		if err := encoder.Encode(event); err != nil {
			if err != io.EOF {
				log.Log.Error(err, "")
			}
			break
		}
	}
	return buf, nil
}
