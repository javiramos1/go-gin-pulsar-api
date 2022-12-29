package pulsar

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	log "github.com/sirupsen/logrus"
	"ramos.com/go-gin-pulsar-api/pkg/metrics"
	"ramos.com/go-gin-pulsar-api/pkg/model"
)

var ctx = context.Background()

// TTL Time to live for cache
const TTL = 60 * time.Minute

type IndexService interface {
	IndexDocument(item *model.IngestionData)
	Close() error
	HealthCheck() error
}

type PulsarClient struct {
	client       *pulsar.Client
	producer     *pulsar.Producer
	topic        string
	retryChannel chan *model.IngestionData
	maxRetries   int
}

func (c *PulsarClient) retry() {

	for doc := range c.retryChannel {
		log.Infof("Retrying Doc %s", doc.Uuid)
		if doc.Retries > c.maxRetries {
			log.Fatalf("Fatal Error Index Doc. Retries: %v", doc.Retries)
			metrics.Error.Inc()
		}
		time.Sleep(time.Duration(doc.Retries*2) * time.Millisecond)
		doc.Retries = doc.Retries + 1
		c.IndexDocument(doc)
	}
}

func readSchema(file string) (string, error) {
	log.Infof("Reading file: %s", file)
	schema, err := ioutil.ReadFile(file)
	if err != nil {
		absPath, _ := filepath.Abs(file)
		log.Infof("Reading file Abs: %s", absPath)
		schema, err = ioutil.ReadFile(absPath)
	}

	if err != nil {
		return "", fmt.Errorf("Error reading file %s: %v", file, err)
	}

	return string(schema), nil
}

// New Creates new Service
func New(connection string, retries int, topic string, retryChannelSize int, retryChannelBuffer int, maxPendingMsg int, schemaPath string) (IndexService, error) {

	log.Infof("New, connection %s", connection)

	schema, err := readSchema(schemaPath)
	if err != nil {
		return nil, err
	}

	client, producer, e := connectPulsarWithRetry(retries, 5*time.Second, connection, &schema, topic, maxPendingMsg)
	if e != nil {
		return nil, e
	}

	retryChannel := make(chan *model.IngestionData, retryChannelBuffer)

	c := &PulsarClient{
		client:   client,
		producer: producer,
		topic:    topic,

		retryChannel: retryChannel,
		maxRetries:   retries,
	}

	for routineNr := int64(0); routineNr < int64(retryChannelSize); routineNr++ {
		go c.retry()
	}

	return c, nil

}

func connectPulsar(connection string, schema *string, topic string, maxPendingMessages int) (pulsar.Client, pulsar.Producer, error) {

	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:                     connection,
		OperationTimeout:        90 * time.Second,
		ConnectionTimeout:       60 * time.Second,
		MaxConnectionsPerBroker: 10,
	})

	avroSchema := pulsar.NewAvroSchema(*schema, nil)

	if err == nil {
		producer, err := client.CreateProducer(pulsar.ProducerOptions{
			Topic:                   topic,
			Schema:                  avroSchema,
			CompressionType:         pulsar.LZ4,
			MaxPendingMessages:      maxPendingMessages,
			BatchingMaxPublishDelay: 5 * time.Second,
			BatchingMaxMessages:     5000,
			BatchingMaxSize:         1024 * 1024,
			CompressionLevel:        pulsar.Faster,
		})
		return client, producer, err
	}

	return nil, nil, err

}

func connectPulsarWithRetry(attempts int, sleep time.Duration, connection string, schema *string, topic string, maxPendingMsg int) (c *pulsar.Client, p *pulsar.Producer, err error) {
	for i := 0; i < attempts; i++ {
		if i > 0 {
			log.Warnf("retrying after error: %v, Attempt: %v", err, i)
			time.Sleep(sleep)
			sleep *= 2
		}
		c, p, err := connectPulsar(connection, schema, topic, maxPendingMsg)
		if err == nil {
			return &c, &p, nil
		}
	}
	return nil, nil, fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}

func (c *PulsarClient) IndexDocument(doc *model.IngestionData) {
	(*c.producer).SendAsync(ctx, &pulsar.ProducerMessage{
		Value: &doc,
	}, func(messageID pulsar.MessageID, producerMessage *pulsar.ProducerMessage, err error) {
		if err != nil {
			metrics.Error.Inc()
			log.Errorf("Error Index Doc %s. Retries: %v. ERROR: %v", messageID, doc.Retries, err)
			c.retryChannel <- doc // go back to retry
		} else {
			log.Debugf("Index Doc: %s", messageID)
			metrics.DataIngested.Inc()
		}
	})
}

// Close connections
func (c *PulsarClient) Close() error {
	if c.client != nil {
		(*c.client).Close()
	}
	if c.producer != nil {
		(*c.client).Close()
	}
	return nil
}

func (c *PulsarClient) HealthCheck() error {
	if c.client != nil {
		_, e := (*c.client).TopicPartitions(c.topic)
		return e
	}
	return nil
}
