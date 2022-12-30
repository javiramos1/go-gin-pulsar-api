# Go REST API using GIN framework with Apache Pulsar

This is a **GO** demo application that showcases the following features:

- Implement a REST en point using the [GIN](https://gin-gonic.com/) framework.
- Using [OpenAPI](https://www.openapis.org/) and the Golang [code generators](https://github.com/deepmap/oapi-codegen)
- Integration with [Apache Pulsar](https://pulsar.apache.org/) by streaming items from a POST request into Pulsar.
- The use of Pulsar, GIN and custom [Prometheus Metrics](https://prometheus.io/)
- The use of **Avro** with Pulsar. 

The application receives an array of items as a payload and it streams the HTTP POST payload into Pulsar.

The goal is to show how to stream data over HTTP into pulsar using Go channels.

It provider automatic retries, self healing, metrics and high scalability. It can be horizontally scalable.

You can run many workers per producers and create many producers per service.

A companion project which consumes the messages is located [**here**](https://github.com/javiramos1/go-pulsar-elasticsearch).

Metrics are exposed on port **8001** path `/metrics`.

## Goals

- `make db`: Starts Pulsar
- `make stop`: Stops Pulsar
- `make build`: To build docker image
- `make run`: Runs docker image, default port is 8000
- `make push`: Pushes image to the docker registry.

## Env Vars

- `PORT`: Port to listen on, defaults to 8000.
- `PULSAR_URL`: URL for pulsar including port. Example: `pulsar://172.25.0.2:6650` (note that this url is relative to the container where the API application runs; if your pulsar instance is running on localhost this url should be set to the ip of the container where pulsar is running)
- `NUMBER_PRODUCERS`: Number of Pulsar producers to create per topic. This service can scale vertically by increasing this value which will use more memory and CPU, or by increasing the replicas in Kubernetes. Defaults to 1.
- `PRODUCER_CHANNEL_SIZE`: Size of the producer channel, this is the maximum number of messages to hold in memory before start blocking HTTP calls. This allows for buffering in case of a slow consumer and allow time for K8s HPA to increase the number of replicas. Defaults to 1000.
- `DATA_SCHEMA_PATH`: Avro Schema Path
- `INDEX_TOPIC`: Topic name
- `RETRIES`: number of retries for Pulsar

## API

The Open API spec can be found [**here**](schema/rest/openApi.yaml).

In order to stream data, the inputs are arrays of objects.

### Examples

#### data

URL: `http://localhost:8000/data`

```
[
    {
        "identifier": "path",
        "name": "name",
        "type": "type",
        "tags": [
            {
                "type": "is_numeric",
                "value": "false"
            },
             {
                "type": "test",
                "value": "test"
            }
        ]
    }
]
```



#### Model

```
type IngestionRequest struct {
	Tags       []Tag  `json:"tags,omitempty"`
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
	Type       string `json:"type"`
}
```

## Schemas

### Pulsar Topic

Contains the data to be inserted into Pulsar.

```
{
  "name": "IngestionData",
  "type": "record",
  "namespace": "com.ramos",
  "fields": [
    {
      "name": "identifier",
      "type": "string"
    },
    {
      "name": "name",
      "type": "string"
    },
    {
      "name": "uuid",
      "type": "string"
    },
    {
      "name": "type",
      "type": "string"
    },
    {
      "name": "ingestion_time",
      "type": "long"
	  },
    {
      "name": "tags",
      "type": [
        "null",
        {
          "type": "array",
          "items": {
            "name": "Tags",
            "type": "record",
            "namespace": "com.ramos",
            "fields": [
              {
                "name": "type",
                "type": "string"
              },
              {
                "name": "value",
                "type": "string"
              }
            ]
          }
        }
      ],
      "default": null
    }
  ]
}
```
