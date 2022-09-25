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
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/influxdata/influxdb/models"
	json "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

/*
Handle incoming data in Influx's Line protocol
e.g.
weather,location=us-midwest temperature=82 1465839830100400200
Becomes:

	{
		"name": "weather",
		"tags": {"location": "us-midwest"},
		"fields": {"temperature": 82},
		"timestamp": 1465839830100400200
	 }

The other potential change is whether or not we're "flipping" single fields for a VictoriaMetrics output
*/
func deserializeInfluxLine(thread int, msg []byte, flipSingleField bool) []InfluxMetric {
	var outputStats []InfluxMetric
	ProcTimeStart := time.Now()
	// logFields := {"threadNum": thread, "section": "processing"}
	ReceivedMsgs.Inc()
	if msg == nil {
		log.Warning("Empty message received from Kafka")
	} else {
		points, err := models.ParsePoints(msg)
		if err != nil {
			log.WithFields(log.Fields{"threadnum": thread, "error": err, "incoming_msg": msg, "section": "influx Line processing"}).Error("Couldn't process message")
		} else {
			for _, point := range points {
				jsonMsg := InfluxMetric{
					Name: string(point.Name()), Tags: make(map[string]string),
					Fields: make(map[string]interface{}), Timestamp: 0,
				}
				for _, tag := range point.Tags() {
					jsonMsg.Tags[string(tag.Key)] = string(tag.Value)
				}
				fields, err := point.Fields()
				if err != nil {
					log.WithFields(log.Fields{"threadnum": thread, "error": err, "incoming_msg": msg, "section": "influx Line processing"}).Error("No fields in incoming message?")
				}
				if flipSingleField && len(fields) < 2 {
					fieldName := ""
					for field, value := range fields {
						fieldName = field
						jsonMsg.Fields["value"] = value
					}
					jsonMsg.Name += fmt.Sprintf("_%v", fieldName)
				} else {
					for field, value := range fields {
						jsonMsg.Fields[field] = value
					}
				}
				jsonMsg.Timestamp = point.UnixNano()
				outputStats = append(outputStats, jsonMsg)
			}
		}
	}
	ProcessTime.Add(float64(time.Now().Sub(ProcTimeStart)) / TimeSegmentDivisor)
	return outputStats
}

/*
Handle incoming data in Influx's default JSON output...
e.g.

	{
	    "fields": {
	        "field_1": 30,
	        "field_2": 4,
	        "field_N": 59,
	        "n_images": 660
	    },
	    "name": "docker",
	    "tags": {
	        "host": "raynor"
	    },
	    "timestamp": 1458229140
	}

The only potential change is whether or not we're "flipping" single fields for a VictoriaMetrics output
*/
func deserializeInfluxJSON(thread int, msg []byte, flipSingleField bool) []InfluxMetric {
	var outputStats []InfluxMetric
	ProcTimeStart := time.Now()
	// logFields := {"threadNum": thread, "section": "processing"}
	ReceivedMsgs.Inc()
	if msg == nil {
		log.Warning("Empty message received from Kafka")
	} else {
		var jsonMsg InfluxMetric
		err := json.Unmarshal(msg, &jsonMsg)
		if err != nil {
			log.WithFields(log.Fields{"threadNum": thread, "error": err, "incoming_msg": msg, "section": "influx JSON processing"}).Error("Couldn't process message")
		} else {
			if flipSingleField && len(jsonMsg.Fields) < 2 {
				// log.WithFields(log.Fields{"Message": jsonMsg, "threadNum": thread, "section": "processing"}).Info("Processing Message")
				/*
					Address single field messages

					We'll add the field name to the metric name and make the field `value`.
					This will make -influxSkipSingleField
				*/
				tmpMetric := InfluxMetric{
					Name: "", Fields: make(map[string]interface{}),
					Tags: make(map[string]string), Timestamp: jsonMsg.Timestamp,
				}
				fieldName := ""
				for key, value := range jsonMsg.Fields {
					fieldName = key
					tmpMetric.Fields["value"] = value
				}
				tmpMetric.Name = fmt.Sprintf("%v_%v", jsonMsg.Name, fieldName)
				for k, v := range jsonMsg.Tags {
					tmpMetric.Tags[k] = v
				}
				// log.WithFields(log.Fields{"Message": tmpMetric, "threadNum": thread, "section": "influx JSON processing"}).Info("Finished Message")
				outputStats = append(outputStats, tmpMetric)
			} else {
				outputStats = append(outputStats, jsonMsg)
			}
		}
	}
	ProcessTime.Add(float64(time.Now().Sub(ProcTimeStart)) / TimeSegmentDivisor)
	return outputStats
}

/*
Handle data in Prometheus' JSON format

	{
	  "timestamp": "1970-01-01T00:00:00Z",
	  "value": "9876543210",
	  "name": "up",

	  "labels": {
	    "__name__": "up",
	    "label1": "value1",
	    "label2": "value2"
	  }
	}

Because Prometheus metrics should already meet the prometheus data model requirements, we don't send them
through filtering. This means we _do_ have to handle normalization here, as well as addressing the "single field"
issue for VictoriaMetrics outputs.
*/
func deserializePromJSON(thread int, msg []byte, normalize bool, flipSingleField bool) []InfluxMetric {
	var outputStats []InfluxMetric
	ProcTimeStart := time.Now()
	ReceivedMsgs.Inc()
	if msg == nil {
		log.Warning("Empty message received from Kafka")
	} else {
		// nice that bytes has a ToLower functions like strings do, makes normalization easy
		if normalize {
			msg = bytes.ToLower(msg)
		}
		var jsonMsg PromMetric
		err := json.Unmarshal(msg, &jsonMsg)
		if err != nil {
			log.WithFields(log.Fields{"threadNum": thread, "error": err, "incoming_msg": msg, "section": "prometheus processing"}).Error("Couldn't process message")
		} else {
			/*
				normalizing the bytes before we serialize means the timestamp field gets slightly munged.
				This is because the timestamp coming in may be in RFC3339
			*/
			if normalize {
				jsonMsg.Timestamp = strings.ToUpper(jsonMsg.Timestamp)
			}
			/*
				Timestamps produced by https://github.com/Telefonica/prometheus-kafka-adapter (which we're relying on)
				come in as RFC3339. Need to convert that back to int64
			*/
			ts, err := time.Parse(time.RFC3339, jsonMsg.Timestamp)
			if err != nil {
				log.WithFields(log.Fields{"threadNum": thread, "error": err, "incoming_msg": msg, "section": "prometheus processing", "timestamp": jsonMsg.Timestamp}).Fatal("Invalid timestamp in message")
			} else {
				/*
					For prometheus metrics, tags/labels don't
					need special processing.
				*/
				finalMsg := InfluxMetric{
					Name: "", Fields: make(map[string]interface{}),
					Tags: make(map[string]string), Timestamp: ts.Unix(),
				}
				for key, value := range jsonMsg.Labels {
					// don't add the __name__ tag, it's the name of the metric already
					if key == "__name__" {
						continue
					}
					finalMsg.Tags[key] = value
				}
				if flipSingleField {
					/*
						all prometheus metrics have a single "field" value
						so handling single field issues is simpler:
						We simply make the single field key `value`.
					*/
					finalMsg.Name = jsonMsg.Name
					finalMsg.Fields["value"] = jsonMsg.Value
				} else {
					/*
						The dance around turning a regular prometheus object
						into an influx object is a bit weirder...
						If we have multiple pieces to our name based on a consistent
						splittable token (_ in our case), we can make the
						field value be the final section of the metric "name".

						If there's only one object after splitting on our token,
						we simply add "value" as the field name.
					*/
					tmpSlice := strings.Split(jsonMsg.Name, "_")
					tmpSliceLen := len(tmpSlice)
					if tmpSliceLen > 1 {
						finalMsg.Fields[tmpSlice[tmpSliceLen-1]] = jsonMsg.Value
						finalMsg.Name = strings.Join(tmpSlice[0:tmpSliceLen-1], "_")
					} else {
						finalMsg.Fields["value"] = jsonMsg.Value
						finalMsg.Name = jsonMsg.Name
					}
				}
				outputStats = append(outputStats, finalMsg)
			}
		}
	}
	ProcessTime.Add(float64(time.Now().Sub(ProcTimeStart)) / TimeSegmentDivisor)
	return outputStats
}

// ProcessInfluxLineMsg : parse and forward an influx line protocol message
func ProcessInfluxLineMsg(ctx context.Context, thread int, inChannel chan []byte, outChannel chan InfluxMetric, wg *sync.WaitGroup, flipSingleField bool) {
	log.WithFields(log.Fields{"threadNum": thread, "section": "influx Line processing"}).Info("processing thread starting...")
	defer wg.Done()

processloop:
	for {
		select {
		case msg := <-inChannel:
			for _, metric := range deserializeInfluxLine(thread, msg, flipSingleField) {
				outChannel <- metric
			}
		case <-ctx.Done():
			log.WithFields(log.Fields{"threadNum": thread, "section": "influx Line processing"}).Info("Closing processing thread...")
			for msg := range inChannel {
				for _, metric := range deserializeInfluxLine(thread, msg, flipSingleField) {
					outChannel <- metric
				}
			}
			break processloop
		}
	}
}

// ProcessInfluxJSONMsg : parse and forward an influx JSON protocol message
func ProcessInfluxJSONMsg(ctx context.Context, thread int, inChannel chan []byte, outChannel chan InfluxMetric, wg *sync.WaitGroup, flipSingleField bool) {
	log.WithFields(log.Fields{"threadNum": thread, "section": "influx JSON processing"}).Info("processing thread starting...")
	defer wg.Done()

processloop:
	for {
		select {
		case msg := <-inChannel:
			for _, metric := range deserializeInfluxJSON(thread, msg, flipSingleField) {
				outChannel <- metric
			}
		case <-ctx.Done():
			log.WithFields(log.Fields{"threadNum": thread, "section": "influx JSON processing"}).Info("Closing processing thread...")
			for msg := range inChannel {
				for _, metric := range deserializeInfluxJSON(thread, msg, flipSingleField) {
					outChannel <- metric
				}
			}
			break processloop
		}
	}
}

// ProcessPromMsg : parse and forward a Prometheus JSON protocol message
func ProcessPromMsg(ctx context.Context, thread int, inChannel chan []byte, outChannel chan InfluxMetric, normalize bool, flipSingleField bool, wg *sync.WaitGroup) {
	log.WithFields(log.Fields{"threadNum": thread, "section": "prometheus processing"}).Info("processing thread starting...")
	defer wg.Done()

processloop:
	for {
		select {
		case msg := <-inChannel:
			for _, metric := range deserializePromJSON(thread, msg, normalize, flipSingleField) {
				outChannel <- metric
			}
		case <-ctx.Done():
			log.WithFields(log.Fields{"threadNum": thread, "section": "prometheus processing"}).Info("Closing processing thread...")
			for msg := range inChannel {
				for _, metric := range deserializePromJSON(thread, msg, normalize, flipSingleField) {
					outChannel <- metric
				}
			}
			break processloop
		}
	}
}
