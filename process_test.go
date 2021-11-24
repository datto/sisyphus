/*This file is part of sisyphus.
 *
 * Copyright Datto, Inc.
 * Author: John Seekins <jseekins@datto.com>
 *
 * Licensed under the GNU General Public License Version 3
 * Fedora-License-Identifier: GPLv3+
 * SPDX-2.0-License-Identifier: GPL-3.0+
 * SPDX-3.0-License-Identifier: GPL-3.0-or-later
 *
 * sisyphus is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * sisyphus is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with sisyphus.  If not, see <https://www.gnu.org/licenses/>.
 */
package main

import (
	"testing"
)

func TestInfluxLine(t *testing.T) {
	var results []InfluxMetric
	msg := "test_metric,tag=Value field=1 1637090544726635243"
	results = deserializeInfluxLine(1, []byte(msg), false)
	if results[0].Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be '1637090544726635243'", results[0].Timestamp)
	}
	if results[0].Name != "test_metric" {
		t.Fatalf("Name (without single field flip) is wrong: %v -> should be 'test_metric'", results[0].Name)
	}
	if len(results[0].Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results[0].Fields))
	}
	if results[0].Fields["field"].(float64) != 1 {
		t.Fatalf("field value is wrong: %v -> should be '1'", results[0].Fields["field"])
	}
	if len(results[0].Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results[0].Tags))
	}
	if results[0].Tags["tag"] != "Value" {
		t.Fatalf("tag value is wrong: %v -> should be 'Value'", results[0].Tags["tag"])
	}
	results = deserializeInfluxLine(1, []byte(msg), true)
	if results[0].Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be '1637090544726635243'", results[0].Timestamp)
	}
	if results[0].Name != "test_metric_field" {
		t.Fatalf("Name (with single field flip) is wrong: %v -> should be 'test_metric_field'", results[0].Name)
	}
	if len(results[0].Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results[0].Fields))
	}
	if results[0].Fields["value"].(float64) != 1 {
		t.Fatalf("field value is wrong: %v -> should be '1'", results[0].Fields["value"])
	}
	if len(results[0].Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results[0].Tags))
	}
	if results[0].Tags["tag"] != "Value" {
		t.Fatalf("tag value is wrong: %v -> should be 'Value'", results[0].Tags["tag"])
	}
	// space after comma and before tags
	msg = "test_metric, tag=value field=1 1637090544726635243"
	results = deserializeInfluxLine(1, []byte(msg), false)
	if len(results) > 0 {
		t.Fatalf("Improperly formatted line protocol emitted data? %v", results)
	}
}

func TestInfluxJSON(t *testing.T) {
	var results []InfluxMetric
	msg := "{\"fields\": {\"field\": 1}, \"tags\": {\"tag\": \"Value\"}, \"name\": \"test_metric\", \"timestamp\": 1637090544726635243}"
	results = deserializeInfluxJSON(1, []byte(msg), false)
	if results[0].Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be '1637090544726635243'", results[0].Timestamp)
	}
	if results[0].Name != "test_metric" {
		t.Fatalf("Name (without single field flip) is wrong: %v -> should be 'test_metric'", results[0].Name)
	}
	if len(results[0].Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results[0].Fields))
	}
	if results[0].Fields["field"].(float64) != 1 {
		t.Fatalf("field value is wrong: %v -> should be '1'", results[0].Fields["field"])
	}
	if len(results[0].Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results[0].Tags))
	}
	if results[0].Tags["tag"] != "Value" {
		t.Fatalf("tag value is wrong: %v -> should be 'Value'", results[0].Tags["tag"])
	}
	results = deserializeInfluxJSON(1, []byte(msg), true)
	if results[0].Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be '1637090544726635243'", results[0].Timestamp)
	}
	if results[0].Name != "test_metric_field" {
		t.Fatalf("Name (with single field flip) is wrong: %v -> should be 'test_metric_field'", results[0].Name)
	}
	if len(results[0].Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results[0].Fields))
	}
	if results[0].Fields["value"].(float64) != 1 {
		t.Fatalf("field value is wrong: %v -> should be '1'", results[0].Fields["value"])
	}
	if len(results[0].Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results[0].Tags))
	}
	if results[0].Tags["tag"] != "Value" {
		t.Fatalf("tag value is wrong: %v -> should be 'Value'", results[0].Tags["tag"])
	}
	// missing comma after `fields` object
	msg = "{\"fields\": {\"field\": 1} \"tags\": {\"tag\": \"value\"}, \"name\": \"test_metric\", \"timestamp\": 1637090544726635243}"
	results = deserializeInfluxJSON(1, []byte(msg), false)
	if len(results) > 0 {
		t.Fatalf("Improperly formatted influx JSON emitted data? %v", results)
	}
}

func TestPrometheusJSON(t *testing.T) {
	var results []InfluxMetric
	msg := "{\"value\": \"2\", \"name\": \"test_metric_field\", \"timestamp\": \"2021-11-16T07:20:50.52Z\", \"labels\": {\"__name__\": \"test_metric_field\", \"tag\": \"Value\"}}"
	results = deserializePromJSON(1, []byte(msg), false, false)
	// timestamp gets smooshed down to seconds in parsing
	if results[0].Timestamp != 1637047250 {
		t.Fatalf("Timestamp is invalid: %v -> should be '1637047250'", results[0].Timestamp)
	}
	if results[0].Name != "test_metric" {
		t.Fatalf("Name (without single field flip) is wrong: %v -> should be 'test_metric'", results[0].Name)
	}
	if len(results[0].Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results[0].Fields))
	}
	if results[0].Fields["field"].(string) != "2" {
		t.Fatalf("field value is wrong: %v -> should be '2'", results[0].Fields["field"])
	}
	if len(results[0].Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results[0].Tags))
	}
	if results[0].Tags["tag"] != "Value" {
		t.Fatalf("tag value is wrong: %v -> should be 'Value'", results[0].Tags["tag"])
	}
	results = deserializePromJSON(1, []byte(msg), false, true)
	if results[0].Timestamp != 1637047250 {
		t.Fatalf("Timestamp is invalid: %v -> should be '1637047250'", results[0].Timestamp)
	}
	if results[0].Name != "test_metric_field" {
		t.Fatalf("Name (with single field flip) is wrong: %v -> should be 'test_metric_field'", results[0].Name)
	}
	if len(results[0].Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results[0].Fields))
	}
	if results[0].Fields["value"].(string) != "2" {
		t.Fatalf("field value is wrong: %v -> should be '2'", results[0].Fields["value"])
	}
	if len(results[0].Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results[0].Tags))
	}
	if results[0].Tags["tag"] != "Value" {
		t.Fatalf("tag value is wrong: %v -> should be 'Value'", results[0].Tags["tag"])
	}
	msg = "{\"value\": \"2\", \"name\": \"test_metric_field\", \"timestamp\": \"2021-11-16T07:20:50.52Z\", \"labels\": {\"__name__\": \"test_metric_field\", \"tag\": \"Value\"}}"
	results = deserializePromJSON(1, []byte(msg), true, false)
	if results[0].Timestamp != 1637047250 {
		t.Fatalf("Timestamp is invalid: %v -> should be '1637047250'", results[0].Timestamp)
	}
	if results[0].Name != "test_metric" {
		t.Fatalf("Name (without single field flip) is wrong: %v -> should be 'test_metric'", results[0].Name)
	}
	if len(results[0].Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results[0].Fields))
	}
	if results[0].Fields["field"].(string) != "2" {
		t.Fatalf("field value is wrong: %v -> should be '2'", results[0].Fields["field"])
	}
	if len(results[0].Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results[0].Tags))
	}
	if results[0].Tags["tag"] != "value" {
		t.Fatalf("tag value is wrong: %v -> should be 'value'", results[0].Tags["tag"])
	}
	results = deserializePromJSON(1, []byte(msg), true, true)
	if results[0].Timestamp != 1637047250 {
		t.Fatalf("Timestamp is invalid: %v -> should be '1637047250'", results[0].Timestamp)
	}
	if results[0].Name != "test_metric_field" {
		t.Fatalf("Name (with single field flip) is wrong: %v -> should be 'test_metric_field'", results[0].Name)
	}
	if len(results[0].Fields) != 1 {
		t.Fatalf("field count is wrong: %v -> should be 1", len(results[0].Fields))
	}
	if results[0].Fields["value"].(string) != "2" {
		t.Fatalf("field value is wrong: %v -> should be '2'", results[0].Fields["value"])
	}
	if len(results[0].Tags) != 1 {
		t.Fatalf("tag count is wrong: %v -> should be 1", len(results[0].Tags))
	}
	if results[0].Tags["tag"] != "value" {
		t.Fatalf("tag value is wrong: %v -> should be 'value'", results[0].Tags["tag"])
	}
	// no comma after value
	msg = "{\"value\": \"2\" \"timestamp\": 1637090544726635243, \"labels\": {\"__name__\": \"test_metric\", \"tag\": \"value\"}}"
	results = deserializePromJSON(1, []byte(msg), false, false)
	if len(results) > 0 {
		t.Fatalf("Improperly formatted Prometheus JSON emitted data? %v", results)
	}
}
