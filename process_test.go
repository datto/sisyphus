package main

import (
	"testing"
)

func TestInfluxLine(t *testing.T) {
	var results []InfluxMetric
	msg := "test_metric,tag=value field=1 1637090544726635243"
	results = deserializeInfluxLine(1, []byte(msg), false)
	if results[0].Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be 1637090544726635243", results[0].Timestamp)
	}
	if results[0].Name != "test_metric" {
		t.Fatalf("Name (without single field flip) is wrong: %v -> should be test_metric", results[0].Name)
	}
	if len(results[0].Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results[0].Fields))
	}
	if results[0].Fields["field"] != 1 {
		t.Fatalf("field value is wrong: %v -> should be 1", results[0].Fields["field"])
	}
	if len(results[0].Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results[0].Tags))
	}
	if results[0].Tags["tag"] != "value" {
		t.Fatalf("tag value is wrong: %v -> should be value", results[0].Tags["tag"])
	}
	results = deserializeInfluxLine(1, []byte(msg), true)
	if results[0].Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be 1637090544726635243", results[0].Timestamp)
	}
	if results[0].Name != "test_metric_field" {
		t.Fatalf("Name (with single field flip) is wrong: %v -> should be test_metric_field", results[0].Name)
	}
	if len(results[0].Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results[0].Fields))
	}
	if results[0].Fields["value"] != 1 {
		t.Fatalf("field value is wrong: %v -> should be 1", results[0].Fields["value"])
	}
	if len(results[0].Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results[0].Tags))
	}
	if results[0].Tags["tag"] != "value" {
		t.Fatalf("tag value is wrong: %v -> should be value", results[0].Tags["tag"])
	}
	msg = "test_metric, tag=value field=1 1637090544726635243"
	results = deserializeInfluxLine(1, []byte(msg), false)
	if len(results) > 0 {
		t.Fatalf("Improperly formatted line protocol emitted data? %v", results)
	}
}

func TestInfluxJSON(t *testing.T) {
	var results []InfluxMetric
	msg := "{\"fields\": {\"field\": 1}, \"tags\": {\"tag\": \"value\"}, \"name\": \"test_metric\", \"timestamp\": 1637090544726635243}"
	results = deserializeInfluxJSON(1, []byte(msg), false)
	if results[0].Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be 1637090544726635243", results[0].Timestamp)
	}
	if results[0].Name != "test_metric" {
		t.Fatalf("Name (without single field flip) is wrong: %v -> should be test_metric", results[0].Name)
	}
	if len(results[0].Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results[0].Fields))
	}
	if results[0].Fields["field"] != 1 {
		t.Fatalf("field value is wrong: %v -> should be 1", results[0].Fields["field"])
	}
	if len(results[0].Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results[0].Tags))
	}
	if results[0].Tags["tag"] != "value" {
		t.Fatalf("tag value is wrong: %v -> should be value", results[0].Tags["tag"])
	}
	results = deserializeInfluxJSON(1, []byte(msg), true)
	if results[0].Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be 1637090544726635243", results[0].Timestamp)
	}
	if results[0].Name != "test_metric_field" {
		t.Fatalf("Name (with single field flip) is wrong: %v -> should be test_metric_field", results[0].Name)
	}
	if len(results[0].Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results[0].Fields))
	}
	if results[0].Fields["value"] != 1 {
		t.Fatalf("field value is wrong: %v -> should be 1", results[0].Fields["value"])
	}
	if len(results[0].Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results[0].Tags))
	}
	if results[0].Tags["tag"] != "value" {
		t.Fatalf("tag value is wrong: %v -> should be value", results[0].Tags["tag"])
	}
	msg = "{\"fields\": {\"field\": 1} \"tags\": {\"tag\": \"value\"}, \"name\": \"test_metric\", \"timestamp\": 1637090544726635243}"
	results = deserializeInfluxJSON(1, []byte(msg), false)
	if len(results) > 0 {
		t.Fatalf("Improperly formatted influx JSON emitted data? %v", results)
	}
}

func TestPrometheusJSON(t *testing.T) {
	var results []InfluxMetric
	msg := "{\"value\": 2, \"timestamp\": \"2021-11-16T07:20:50.52Z\", \"labels\": {\"__name__\": \"test_metric\", \"tag\": \"value\"}}"
	results = deserializePromJSON(1, []byte(msg), false, false)
	if results[0].Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be 1637090544726635243", results[0].Timestamp)
	}
	if results[0].Name != "test_metric" {
		t.Fatalf("Name (without single field flip) is wrong: %v -> should be test_metric", results[0].Name)
	}
	if len(results[0].Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results[0].Fields))
	}
	if results[0].Fields["field"] != 1 {
		t.Fatalf("field value is wrong: %v -> should be 1", results[0].Fields["field"])
	}
	if len(results[0].Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results[0].Tags))
	}
	if results[0].Tags["tag"] != "value" {
		t.Fatalf("tag value is wrong: %v -> should be value", results[0].Tags["tag"])
	}
	results = deserializePromJSON(1, []byte(msg), false, true)
	if results[0].Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be 1637090544726635243", results[0].Timestamp)
	}
	if results[0].Name != "test_metric_field" {
		t.Fatalf("Name (with single field flip) is wrong: %v -> should be test_metric_field", results[0].Name)
	}
	if len(results[0].Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results[0].Fields))
	}
	if results[0].Fields["value"] != 1 {
		t.Fatalf("field value is wrong: %v -> should be 1", results[0].Fields["value"])
	}
	if len(results[0].Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results[0].Tags))
	}
	if results[0].Tags["tag"] != "value" {
		t.Fatalf("tag value is wrong: %v -> should be value", results[0].Tags["tag"])
	}
	msg = "{\"value\": 2, \"timestamp\": \"2021-11-16T07:20:50.52Z\", \"labels\": {\"__name__\": \"test_metric\", \"tag\": \"Value\"}}"
	results = deserializePromJSON(1, []byte(msg), true, false)
	if results[0].Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be 1637090544726635243", results[0].Timestamp)
	}
	if results[0].Name != "test_metric" {
		t.Fatalf("Name (without single field flip) is wrong: %v -> should be test_metric", results[0].Name)
	}
	if len(results[0].Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results[0].Fields))
	}
	if results[0].Fields["field"] != 1 {
		t.Fatalf("field value is wrong: %v -> should be 1", results[0].Fields["field"])
	}
	if len(results[0].Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results[0].Tags))
	}
	if results[0].Tags["tag"] != "value" {
		t.Fatalf("tag value is wrong: %v -> should be value", results[0].Tags["tag"])
	}
	results = deserializePromJSON(1, []byte(msg), true, true)
	if results[0].Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be 1637090544726635243", results[0].Timestamp)
	}
	if results[0].Name != "test_metric_field" {
		t.Fatalf("Name (with single field flip) is wrong: %v -> should be test_metric_field", results[0].Name)
	}
	if len(results[0].Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results[0].Fields))
	}
	if results[0].Fields["value"] != 1 {
		t.Fatalf("field value is wrong: %v -> should be 1", results[0].Fields["value"])
	}
	if len(results[0].Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results[0].Tags))
	}
	if results[0].Tags["tag"] != "value" {
		t.Fatalf("tag value is wrong: %v -> should be value", results[0].Tags["tag"])
	}

	msg = "{\"value\": 2 \"timestamp\": 1637090544726635243, \"labels\": {\"__name__\": \"test_metric\", \"tag\": \"value\"}}"
	results = deserializePromJSON(1, []byte(msg), false, false)
	if len(results) > 0 {
		t.Fatalf("Improperly formatted Prometheus JSON emitted data? %v", results)
	}
}
