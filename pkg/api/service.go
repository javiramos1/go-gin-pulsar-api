package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"ramos.com/go-gin-pulsar-api/pkg/api_model"
	"ramos.com/go-gin-pulsar-api/pkg/metrics"
	"ramos.com/go-gin-pulsar-api/pkg/model"
	"ramos.com/go-gin-pulsar-api/pkg/pulsar"
)

const (
	ContentType     = "content-type"
	ContentTypeJson = "application/json"
)

type IngestionServiceParams struct {
	PulsarUrl           string
	Producers           int
	ProducerChannelSize int
	Retries             int
	IndexTopic          string
	ThreadsPerProducer  int
	Schema              string
}

type IngestionService struct {
	parameters  *IngestionServiceParams
	dataChannel chan *model.IngestionRequest
	Producers   []*pulsar.IndexService
}

func NewClient(parameters *IngestionServiceParams) (IngestionService, error) {

	log.Infof("NewClient parameters: %s", parameters)

	dataChannel := make(chan *model.IngestionRequest, parameters.ProducerChannelSize)

	producers := []*pulsar.IndexService{}

	for p := int64(0); p < int64(parameters.Producers); p++ {

		log.Info("Creating Topic Client %v", p)
		producer, err := pulsar.New(parameters.PulsarUrl, parameters.Retries, parameters.IndexTopic,
			parameters.ThreadsPerProducer, parameters.ProducerChannelSize, parameters.ProducerChannelSize, parameters.Schema)
		if err != nil {
			handleError(err)
		}
		time.Sleep(1 * time.Second)

		producers = append(producers, &producer)

		for threadNr := int64(0); threadNr < int64(parameters.ThreadsPerProducer); threadNr++ {
			log.Debugf("Creating Thread %d", threadNr)
			go processDataRoutine(dataChannel, &producer)
		}

	}

	return IngestionService{
		parameters:  parameters,
		dataChannel: dataChannel,
		Producers:   producers,
	}, nil

}

func processDataRoutine(dataChannel chan *model.IngestionRequest, producer *pulsar.IndexService) {

	for data := range dataChannel {
		log.Debugf("processDataRoutine %v", data)

		indexData := data.Parse()

		(*producer).IndexDocument(&indexData)
	}
}

func (t IngestionService) IngestData(c *gin.Context) {
	log.Infof("Ingest Item")
	processJsonBody(c, func(dec *json.Decoder) {
		var i model.IngestionRequest
		err := dec.Decode(&i)
		if err != nil {
			sendError(c, http.StatusBadRequest, "Error parsing Data")
		}

		t.dataChannel <- &i
	})
	c.JSON(http.StatusOK, api_model.IngestResponse{
		Received: true,
	})
}

func processJsonBody(c *gin.Context, f func(d *json.Decoder)) {
	dec := json.NewDecoder(c.Request.Body)

	// read open bracket
	_, err := dec.Token()
	if err != nil {
		sendError(c, http.StatusBadRequest, "Error parsing input array")
	}

	// while the array contains values
	for dec.More() {
		f(dec)
	}

	// read closing bracket
	_, err = dec.Token()
	if err != nil {
		log.Errorf("Error parsing closing token")
		handleError(err)
	}

}

func sendError(c *gin.Context, code int, message string) {
	petErr := api_model.Error{
		Code:    int32(code),
		Message: message,
	}
	handleError(nil)
	c.JSON(code, petErr)
}

func handleError(err error) {
	log.Errorf("Error: %s", err)
	metrics.Error.Inc()
}
