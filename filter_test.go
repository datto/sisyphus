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

/*
Things we should check:
1. metric where nothing changes
2. metric with bad name
3. metric with no fields
4. normalized metrics (lower cased)
5. metric names with "bad" characters
6. metric tag names with "bad" characters
7. metric field names with "bad" characters
*/
func TestFilter(t *testing.T) {
	var err error
	var results InfluxMetric
	// nothing should change here
	msg := InfluxMetric{Name: "test_metric", Fields: map[string]interface{}{"field": float64(1)},
		Tags: map[string]string{"tag": "Value"}, Timestamp: 1637090544726635243}
	results, err = filterMsg(1, msg, false)
	if err != nil {
		t.Fatalf("Failed to parse valid metric: %v -> %v", msg, err)
	}
	if results.Timestamp != 1637090544726635243 {
		t.Fatalf("Timestamp is invalid: %v -> should be '1637090544726635243'", results.Timestamp)
	}
	if results.Name != "test_metric" {
		t.Fatalf("Name is wrong: %v -> should be 'test_metric'", results.Name)
	}
	if results.Fields["field"].(float64) != 1 {
		t.Fatalf("field value is wrong: %v -> should be '1'", results.Fields["field"])
	}
	if results.Tags["tag"] != "Value" {
		t.Fatalf("tag value is wrong: %v -> should be 'Value'", results.Tags["tag"])
	}
	// check for bad metric name
	msg = InfluxMetric{Name: "_test_metric", Fields: map[string]interface{}{"field": 1},
		Tags: map[string]string{"tag": "Value"}, Timestamp: 1637090544726635243}
	results, err = filterMsg(1, msg, false)
	if err == nil {
		t.Fatalf("Received results from a bad metric name! Impossible! %v", results)
	}
	// check for metric with no fields
	msg = InfluxMetric{Name: "test_metric", Fields: map[string]interface{}{},
		Tags: map[string]string{"tag": "Value"}, Timestamp: 1637090544726635243}
	results, err = filterMsg(1, msg, false)
	if err == nil {
		t.Fatalf("Received results from empty field list! Impossible! %v", results)
	}
	// check for normalization
	msg = InfluxMetric{Name: "test_metric", Fields: map[string]interface{}{"field": 1},
		Tags: map[string]string{"tag": "Value"}, Timestamp: 1637090544726635243}
	results, err = filterMsg(1, msg, true)
	if err != nil {
		t.Fatalf("Failed to parse valid metric: %v -> %v", msg, err)
	}
	if results.Tags["tag"] != "value" {
		t.Fatalf("Normalization failed: %v", results)
	}
	// bad characters
	msg = InfluxMetric{Name: "test-metric", Fields: map[string]interface{}{"field.1": 1},
		Tags: map[string]string{"tag 1": "Value"}, Timestamp: 1637090544726635243}
	results, err = filterMsg(1, msg, false)
	if err != nil {
		t.Fatalf("Failed to parse valid metric: %v -> %v", msg, err)
	}
	if results.Name != "test_metric" {
		t.Fatalf("Failed to reformat bad metric name %v -> should be 'test_metric'", results.Name)
	}
	if _, ok := results.Fields["field_1"]; !ok {
		t.Fatalf("Failed to reformat bad field name %v -> should be 'field_1'", results.Fields)
	}
	if _, ok := results.Tags["tag_1"]; !ok {
		t.Fatalf("Failed to reformat bad tag name %v -> should be 'tag_1'", results.Tags)
	}
}
