package main

import (
	"fmt"
	"testing"
	"time"
)

func TestInfluxLine(t *testing.T) {
	outChan := make(chan InfluxMetric, 1)
	msg := "test_metric,tag=value field=1 1637090544726635243"
	deserilaizeInfluxLine(1, []byte(msg), false, outChan)

	msg := "test_metric,tag=value field=1 1637090544726635243"
	deserilaizeInfluxLine(1, []byte(msg), true, outChan)

	msg := "test_metric, tag=value field=1 1637090544726635243"
	deserilaizeInfluxLine(1, []byte(msg), false, outChan)
}

func TestInfluxJSON(t *testing.T) {
	outChan := make(chan InfluxMetric, 1)
	msg := "{\"fields\": {\"field\": 1}, \"tags\": {\"tag\": \"value\"}, \"name\": \"test_metric\", \"timestamp\": 1637090544726635243}"
}

func TestPrometheusJSON(t *testing.T) {
	outChan := make(chan InfluxMetric, 1)
	msg := "{\"value\": 2, \"timestamp\": 1637090544726635243, \"labels\": {\"__name__\": \"test_metric\", \"tag\": \"value\"}}"
}
