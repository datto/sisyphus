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
	"fmt"
	"net/http"
	"time"

	"github.com/VictoriaMetrics/metrics"
	log "github.com/sirupsen/logrus"
)

var (
	startTime = time.Now()
	//DroppedMsgs : Messages dropped during filtering
	DroppedMsgs = metrics.NewCounter("dropped_msg_total")
	//IngestMsgs :  Messages collected from Kafka
	IngestMsgs = metrics.NewCounter("kafka_msg_total")
	//FailedMsgs : Messages dropped during write
	FailedMsgs = metrics.NewCounter("failed_msg_total")
	//FilterSteps : Messages filtered in some way during ingest
	FilterSteps = metrics.NewCounter("filtered_msg_total")
	//MetricsCounted : Individual metrics counted in an incoming message
	MetricsCounted = metrics.NewCounter("metrics_total")
	//ReceivedMsgs : Messages successfully deserialized from Kafka
	ReceivedMsgs = metrics.NewCounter("received_msg_total")
	//SentMsgs : Messages sent to Influx/VictoriaMetrics endpoint
	SentMsgs = metrics.NewCounter("sent_msg_total")
	//FailedWriteTime : Time spent writing data to dead letter queue
	FailedWriteTime = metrics.NewFloatCounter("failed_write_time_secs_total")
	//FilterTime : Time spent filtering data
	FilterTime = metrics.NewFloatCounter("filter_time_secs_total")
	//IngestTime : Time spent collecting data from Kafka
	IngestTime = metrics.NewFloatCounter("ingest_time_secs_total")
	//OutputTime : Time spent writing data to Influx
	OutputTime = metrics.NewFloatCounter("output_time_secs_total")
	//ProcessTime : Time spent deserializing data from Kafka
	ProcessTime = metrics.NewFloatCounter("process_time_secs_total")

	//Uptime : since app start
	Uptime = metrics.NewGauge("app_uptime_secs_total",
		func() float64 {
			return float64(time.Since(startTime).Seconds())
		})
	// Current dead letter queue length
	deadLetterQueueLen = metrics.NewGauge("dead_letter_queue_len",
		func() float64 {
			failedWrites := 0
			for i := 0; i < len(Endpoints); i++ {
				failedWrites += len(Endpoints[i].FailedWritesChan)
			}
			return float64(failedWrites)
		})
	// Current queue length for messages to be filtered
	filterQueueLen = metrics.NewGauge("filter_queue_len",
		func() float64 {
			filtered := 0
			for i := 0; i < len(Endpoints); i++ {
				filtered += len(Endpoints[i].FilterTagChan)
			}
			return float64(filtered)
		})
	// Current write queue length
	writeQueueLen = metrics.NewGauge("output_queue_len",
		func() float64 {
			output := 0
			for i := 0; i < len(Endpoints); i++ {
				output += len(Endpoints[i].OutputTSDBChan)
			}
			return float64(output)
		})
	// Current cached influx-style metrics queue length
	influxIngestQueueLen = metrics.NewGauge("influx_ingest_queue_len",
		func() float64 {
			influxIn := 0
			for i := 0; i < len(Endpoints); i++ {
				influxIn += len(Endpoints[i].ProcessInfluxJSONChan)
			}
			return float64(influxIn)
		})
	// Current cached prometheus-style metrics queue length
	promIngestQueuLen = metrics.NewGauge("prometheus_ingest_queue_len",
		func() float64 {
			promIn := 0
			for i := 0; i < len(Endpoints); i++ {
				promIn += len(Endpoints[i].ProcessPromJSONChan)
			}
			return float64(promIn)
		})
)

//StatsListener : Actually expose an endpoint for stats to be scraped
func StatsListener(address string, port string) {
	http.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
		metrics.WritePrometheus(w, true)
	})
	err := http.ListenAndServe(fmt.Sprintf("%v:%v", address, port), nil)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Fatal("Couldn't start stats listener")
	}
}
