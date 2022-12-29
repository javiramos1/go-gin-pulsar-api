package main

import (
	"context"
	"fmt"

	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	middleware "github.com/deepmap/oapi-codegen/pkg/gin-middleware"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	prometrics "github.com/slok/go-http-metrics/metrics/prometheus"
	promiddleware "github.com/slok/go-http-metrics/middleware"
	ginmiddleware "github.com/slok/go-http-metrics/middleware/gin"
	"ramos.com/go-gin-pulsar-api/pkg/api"
	"ramos.com/go-gin-pulsar-api/pkg/api_server"
	"ramos.com/go-gin-pulsar-api/pkg/model"
	"ramos.com/go-gin-pulsar-api/pkg/pulsar"
)

var ctx = context.Background()
var lastError *time.Time

const (
	LogLevel        = "LOG_LEVEL"
	LogLevelDefault = "info"
)

func setEnvironment() {
	env := os.Getenv("ENV")

	e := godotenv.Load("/etc/" + env + ".env")
	if e != nil {
		e = godotenv.Load()
	}
	handleError(e, true)

	logLevel, _ := log.ParseLevel(getEnvWithDefault(LogLevel, LogLevelDefault))
	log.SetLevel(logLevel)
	log.SetFormatter(&log.JSONFormatter{})
	log.SetReportCaller(true)

	// concurrency level, use 3x number of vCPU
	runtime.GOMAXPROCS(runtime.NumCPU() * 3)
	log.Debugf("GOMAXPROCS %v", runtime.GOMAXPROCS(-1))
}

func getEnvWithDefault(envKey, def string) string {
	val := os.Getenv(envKey)
	if val == "" {
		return def
	}
	return val
}

func handleError(err error, fatal ...bool) {
	if err != nil {
		log.Errorf("ERROR: %v", err)
		t := time.Now()
		lastError = &t
		if len(fatal) > 0 && fatal[0] {
			panic(err)
		}
	}
}

func main() {
	log.Info("Starting Ingestion API")

	setEnvironment()

	producers, _ := strconv.Atoi(os.Getenv("NUMBER_PRODUCERS"))
	size, _ := strconv.Atoi(os.Getenv("PRODUCER_CHANNEL_SIZE"))
	retries, _ := strconv.Atoi(os.Getenv("RETRIES"))
	threadsPerProducer, _ := strconv.Atoi(os.Getenv("THREADS_PER_PRODUCER"))

	params := &api.IngestionServiceParams{
		PulsarUrl:           os.Getenv("PULSAR_URL"),
		Producers:           producers,
		ProducerChannelSize: size,
		IndexTopic:          os.Getenv("INDEX_TOPIC"),
		ThreadsPerProducer:  threadsPerProducer,
		Retries:             retries,
		Schema:              os.Getenv("DATA_SCHEMA_PATH"),
	}

	ingestionApi, err := api.NewClient(params)
	if err != nil {
		handleError(err, true)
	}

	swagger, err := api_server.GetSwagger()
	if err != nil {
		handleError(err, true)
	}

	promw := promiddleware.New(promiddleware.Config{
		Recorder: prometrics.NewRecorder(prometrics.Config{}),
	})

	client := ingestionApi.Producers[0]
	r := gin.Default()
	r.GET("/health", health(client))
	r.GET("/ready", health(client))
	r.Use(middleware.OapiRequestValidator(swagger))
	r.Use(ginmiddleware.Handler("", promw))
	r = api_server.RegisterHandlers(r, ingestionApi)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "8001"
	}

	srv := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("0.0.0.0:%s", port),
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
	}
	log.Info("Initialization completed, starting server...")

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			handleError(err, true)
		}
	}()

	log.Infof("Server Started listening on port %v", port)

	go func() {
		log.Infof("metrics listening at %s", metricsPort)
		if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", metricsPort), promhttp.Handler()); err != nil {
			handleError(err, true)
		}
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-done
	log.Info("Server Stopped")

	ctx, httpCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		log.Info("Cancelling  HTTP")
		httpCancel()
		for _, p := range ingestionApi.Producers {
			(*p).Close()
		}
	}()

	if err := srv.Shutdown(ctx); err != nil {
		log.Errorf("Server Shutdown Failed:%+v", err)
	}

}

func handleHTTPError(e error, c *gin.Context) bool {
	if e != nil {
		handleError(e, true)
		setError(e, c)
		return true
	}

	return false
}

func setError(e error, c *gin.Context) {
	c.JSON(500, gin.H{
		"error": e.Error(),
	})
}

func health(pulsar *pulsar.IndexService) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		err := (*pulsar).HealthCheck()

		if !handleHTTPError(err, c) {

			goVer := runtime.Version()
			numRoutines := runtime.NumGoroutine()

			lastErrorStr := ""
			if lastError != nil {
				lastErrorStr = string(lastError.String())
			}

			result := model.HealthStatus{
				AppName:          "go-gin-pulsar-api",
				Status:           "UP",
				GoVersion:        goVer,
				NumberOfRoutines: numRoutines,
				LastError:        lastErrorStr,
			}

			c.JSON(200, result)
		}

		// }
	}

	return gin.HandlerFunc(fn)
}
