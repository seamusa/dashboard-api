package database

import (
	"time"
)

type PostgresMetric struct {
	Count      int       `json:"count"`
	Total      float64   `json:"total"`
	Minimum    float64   `json:"minimum"`
	Maximum    float64   `json:"maximum"`
	Average    float64   `json:"average"`
	ResourceID string    `json:"-"`
	Time       time.Time `json:"time"`
	MetricName string    `json:"metricName"`
	TimeGrain  string    `json:"timeGrain"`
}
