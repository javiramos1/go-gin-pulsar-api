package model

import (
	"time"

	"github.com/google/uuid"
)

type Tag struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type IngestionRequest struct {
	Tags       []Tag  `json:"tags,omitempty"`
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
	Type       string `json:"type"`
}

func (i IngestionRequest) Parse() IngestionData {
	data := IngestionData{
		Uuid:          uuid.New().String(),
		IngestionTime: time.Now().UnixMilli(),
		Name:          i.Name,
		Type:          i.Type,
		Identifier:    i.Identifier,
	}
	if len(i.Tags) > 0 {
		data.Tags = map[string][]Tag{"array": i.Tags}
	}
	return data
}

type IngestionData struct {
	Tags          map[string][]Tag `json:"tags,omitempty"`
	Uuid          string           `json:"uuid"`
	Identifier    string           `json:"identifier"`
	Name          string           `json:"name"`
	Type          string           `json:"type"`
	IngestionTime int64            `json:"ingestion_time"`
	Retries       int              `json:"-"`
}

type IngestionRequestData = []IngestionRequest

// HealthStatus response
type HealthStatus struct {
	AppName          string `json:"app_name"`
	Status           string `json:"status"`
	GoVersion        string `json:"go_version"`
	NumberOfRoutines int    `json:"number_of_routines"`
	LastError        string `json:"last_error"`
}
