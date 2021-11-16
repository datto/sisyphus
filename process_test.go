package main

import (
	"testing"
)

func getValues(c <-chan InfluxMetric) InfluxMetric {
	var result InfluxMetric
	for i := range c {
		result = i
	}
	return result
}

func TestInfluxLine(t *testing.T) {
	outChan := make(chan InfluxMetric, 1)
	var results InfluxMetric
	msg := "test_metric,tag=value field=1 1637090544726635243"
	deserializeInfluxLine(1, []byte(msg), false, outChan)
	results = getValues(outChan)
	if results.Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be 1637090544726635243", results.Timestamp)
	}
	if results.Name != "test_metric" {
		t.Fatalf("Name (without single field flip) is wrong: %v -> should be test_metric", results.Name)
	}
	if len(results.Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results.Fields))
	}
	if results.Fields["field"] != 1 {
		t.Fatalf("field value is wrong: %v -> should be 1", results.Fields["field"])
	}
	if len(results.Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results.Tags))
	}
	if results.Tags["tag"] != "value" {
		t.Fatalf("tag value is wrong: %v -> should be value", results.Tags["tag"])
	}
	deserializeInfluxLine(1, []byte(msg), true, outChan)
	results = getValues(outChan)
	if results.Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be 1637090544726635243", results.Timestamp)
	}
	if results.Name != "test_metric_field" {
		t.Fatalf("Name (with single field flip) is wrong: %v -> should be test_metric_field", results.Name)
	}
	if len(results.Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results.Fields))
	}
	if results.Fields["value"] != 1 {
		t.Fatalf("field value is wrong: %v -> should be 1", results.Fields["value"])
	}
	if len(results.Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results.Tags))
	}
	if results.Tags["tag"] != "value" {
		t.Fatalf("tag value is wrong: %v -> should be value", results.Tags["tag"])
	}
	msg = "test_metric, tag=value field=1 1637090544726635243"
	deserializeInfluxLine(1, []byte(msg), false, outChan)
	results = getValues(outChan)
}

func TestInfluxJSON(t *testing.T) {
	var results InfluxMetric
	outChan := make(chan InfluxMetric, 1)
	msg := "{\"fields\": {\"field\": 1}, \"tags\": {\"tag\": \"value\"}, \"name\": \"test_metric\", \"timestamp\": 1637090544726635243}"
	deserializeInfluxJSON(1, []byte(msg), false, outChan)
	results = getValues(outChan)
	if results.Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be 1637090544726635243", results.Timestamp)
	}
	if results.Name != "test_metric" {
		t.Fatalf("Name (without single field flip) is wrong: %v -> should be test_metric", results.Name)
	}
	if len(results.Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results.Fields))
	}
	if results.Fields["field"] != 1 {
		t.Fatalf("field value is wrong: %v -> should be 1", results.Fields["field"])
	}
	if len(results.Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results.Tags))
	}
	if results.Tags["tag"] != "value" {
		t.Fatalf("tag value is wrong: %v -> should be value", results.Tags["tag"])
	}
	deserializeInfluxJSON(1, []byte(msg), true, outChan)
	results = getValues(outChan)
	if results.Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be 1637090544726635243", results.Timestamp)
	}
	if results.Name != "test_metric_field" {
		t.Fatalf("Name (with single field flip) is wrong: %v -> should be test_metric_field", results.Name)
	}
	if len(results.Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results.Fields))
	}
	if results.Fields["value"] != 1 {
		t.Fatalf("field value is wrong: %v -> should be 1", results.Fields["value"])
	}
	if len(results.Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results.Tags))
	}
	if results.Tags["tag"] != "value" {
		t.Fatalf("tag value is wrong: %v -> should be value", results.Tags["tag"])
	}
	msg = "{\"fields\": {\"field\": 1} \"tags\": {\"tag\": \"value\"}, \"name\": \"test_metric\", \"timestamp\": 1637090544726635243}"
	deserializeInfluxJSON(1, []byte(msg), false, outChan)
	results = getValues(outChan)
}

func TestPrometheusJSON(t *testing.T) {
	var results InfluxMetric
	outChan := make(chan InfluxMetric, 1)
	msg := "{\"value\": 2, \"timestamp\": \"2021-11-16T07:20:50.52Z\", \"labels\": {\"__name__\": \"test_metric\", \"tag\": \"value\"}}"
	deserializePromJSON(1, []byte(msg), outChan, false, false)
	results = getValues(outChan)
	if results.Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be 1637090544726635243", results.Timestamp)
	}
	if results.Name != "test_metric" {
		t.Fatalf("Name (without single field flip) is wrong: %v -> should be test_metric", results.Name)
	}
	if len(results.Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results.Fields))
	}
	if results.Fields["field"] != 1 {
		t.Fatalf("field value is wrong: %v -> should be 1", results.Fields["field"])
	}
	if len(results.Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results.Tags))
	}
	if results.Tags["tag"] != "value" {
		t.Fatalf("tag value is wrong: %v -> should be value", results.Tags["tag"])
	}
	deserializePromJSON(1, []byte(msg), outChan, false, true)
	results = getValues(outChan)
	if results.Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be 1637090544726635243", results.Timestamp)
	}
	if results.Name != "test_metric_field" {
		t.Fatalf("Name (with single field flip) is wrong: %v -> should be test_metric_field", results.Name)
	}
	if len(results.Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results.Fields))
	}
	if results.Fields["value"] != 1 {
		t.Fatalf("field value is wrong: %v -> should be 1", results.Fields["value"])
	}
	if len(results.Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results.Tags))
	}
	if results.Tags["tag"] != "value" {
		t.Fatalf("tag value is wrong: %v -> should be value", results.Tags["tag"])
	}
	msg = "{\"value\": 2, \"timestamp\": \"2021-11-16T07:20:50.52Z\", \"labels\": {\"__name__\": \"test_metric\", \"tag\": \"Value\"}}"
	deserializePromJSON(1, []byte(msg), outChan, true, false)
	results = getValues(outChan)
	if results.Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be 1637090544726635243", results.Timestamp)
	}
	if results.Name != "test_metric" {
		t.Fatalf("Name (without single field flip) is wrong: %v -> should be test_metric", results.Name)
	}
	if len(results.Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results.Fields))
	}
	if results.Fields["field"] != 1 {
		t.Fatalf("field value is wrong: %v -> should be 1", results.Fields["field"])
	}
	if len(results.Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results.Tags))
	}
	if results.Tags["tag"] != "value" {
		t.Fatalf("tag value is wrong: %v -> should be value", results.Tags["tag"])
	}
	deserializePromJSON(1, []byte(msg), outChan, true, true)
	results = getValues(outChan)
	if results.Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be 1637090544726635243", results.Timestamp)
	}
	if results.Name != "test_metric_field" {
		t.Fatalf("Name (with single field flip) is wrong: %v -> should be test_metric_field", results.Name)
	}
	if len(results.Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results.Fields))
	}
	if results.Fields["value"] != 1 {
		t.Fatalf("field value is wrong: %v -> should be 1", results.Fields["value"])
	}
	if len(results.Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results.Tags))
	}
	if results.Tags["tag"] != "value" {
		t.Fatalf("tag value is wrong: %v -> should be value", results.Tags["tag"])
	}

	msg = "{\"value\": 2 \"timestamp\": 1637090544726635243, \"labels\": {\"__name__\": \"test_metric\", \"tag\": \"value\"}}"
	deserializePromJSON(1, []byte(msg), outChan, false, false)
	results = getValues(outChan)
}
