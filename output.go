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
	"context"
	"sync"
	"time"

	"github.com/influxdata/influxdb-client-go/v2"
	influxapi "github.com/influxdata/influxdb-client-go/v2/api"
	influxapiwrite "github.com/influxdata/influxdb-client-go/v2/api/write"
	log "github.com/sirupsen/logrus"
)

// OutputMeta : meta data about an output we need to know
type OutputMeta struct {
	Thread       int
	BatchSize    uint
	WriteTimeout uint
	MaxRetries   uint
	FlushSegment float64
	URL          string
	TsdOrg       string
	TsdDbName    string
}

// BatchMeta : meta data about the batches we write to our outputs
type BatchMeta struct {
	Thread        int
	BatchCount    uint
	FlushSegment  float64
	Batch         []*influxapiwrite.Point
	BatchSize     uint
	LastFlushTime time.Time
	WriteAPI      influxapi.WriteAPIBlocking
}

var (
	duration time.Duration
)

func writeBatch(thread int, batch []*influxapiwrite.Point, batchCount uint, writeAPI influxapi.WriteAPIBlocking, failedChan chan string) {
	err := writeAPI.WritePoint(context.Background(), batch...)
	if err != nil {
		log.WithFields(log.Fields{"threadNum": thread, "section": "output", "error": err}).Error("Failed Write")
		for _, badpoint := range batch {
			if len(badpoint.TagList()) < 1 {
				/*
					if the metric has no tags, skip it.
					This is largely to avoid writing empty messages to the dead letter queue
				*/
				continue
			}
			/*
				the metric conversion function requires a time.Duration set, so we'll just use a default ("1us")
			*/
			badstr := influxapiwrite.PointToLineProtocol(badpoint, duration)
			failedChan <- badstr
		}
	} else {
		SentMsgs.Add(int(batchCount))
	}
}

func processOutput(msg InfluxMetric, meta *BatchMeta, failedChan chan string) {
	outputTimeStart := time.Now()
	p := influxdb2.NewPoint(msg.Name, msg.Tags, msg.Fields, time.Unix(msg.Timestamp, 0))

	meta.Batch = append(meta.Batch, p)
	meta.BatchCount++
	/*
		We have two reasons to flush the output buffer:
		1. we have reached the number of objects in the buffer we should flush at (meta.BatchSize)
		  * meta.BatchCount >= meta.BatchSize
		2. it's been longer than meta.FlushTimeSegment since the last time we flushed
		  * float64(outputTimeStart.Sub(meta.LastFlushTime))/TimeSegmentDivisor > meta.FlushSegment
		Both of these options are predicated on there _being_ data to flush (because there's no reason to flush an empty buffer)
	*/
	if meta.BatchCount > 0 && (meta.BatchCount >= meta.BatchSize || float64(outputTimeStart.Sub(meta.LastFlushTime))/TimeSegmentDivisor > meta.FlushSegment) {
		writeBatch(meta.Thread, meta.Batch, meta.BatchCount, meta.WriteAPI, failedChan)
		meta.BatchCount = 0
		meta.Batch = meta.Batch[:0]
		meta.LastFlushTime = time.Now()
	}
	OutputTime.Add(float64(time.Now().Sub(outputTimeStart)) / TimeSegmentDivisor)
}

/*
SendTSDB : wrapper to actually send messages to our configured outputs
All incoming messages should be formatted as influx metrics
*/
func SendTSDB(ctx context.Context, inChannel chan InfluxMetric, failedChan chan string, cfg OutputMeta, wg *sync.WaitGroup) {
	var err error
	log.WithFields(log.Fields{"threadNum": cfg.Thread, "section": "output"}).Info("Output thread starting...")
	defer wg.Done()
	client := influxdb2.NewClientWithOptions(
		cfg.URL, "",
		influxdb2.DefaultOptions().
			SetUseGZip(true).
			SetHTTPRequestTimeout(cfg.WriteTimeout).
			SetMaxRetries(cfg.MaxRetries),
	)
	defer client.Close()
	duration, err = time.ParseDuration("1us")
	if err != nil {
		log.WithFields(log.Fields{"threadNum": cfg.Thread, "section": "output", "error": err}).Fatal("Couln't parse static time.Duration?!")
	}

	// properly scoped variables so multiple threads don't stomp on things
	meta := BatchMeta{Thread: cfg.Thread, BatchCount: 0, FlushSegment: cfg.FlushSegment,
		Batch: make([]*influxapiwrite.Point, 0, cfg.BatchSize*2), BatchSize: cfg.BatchSize,
		LastFlushTime: time.Now(), WriteAPI: client.WriteAPIBlocking(cfg.TsdOrg, cfg.TsdDbName)}

outputloop:
	for {
		select {
		case msg := <-inChannel:
			processOutput(msg, &meta, failedChan)
		case <-ctx.Done():
			log.WithFields(log.Fields{"threadNum": cfg.Thread, "section": "output"}).Info("Closing output thread...")
			// make sure to flush anything in the channel before exiting
			for msg := range inChannel {
				processOutput(msg, &meta, failedChan)
			}
			// one last write after finishing to ensure we don't drop data on the floor
			writeBatch(meta.Thread, meta.Batch, meta.BatchCount, meta.WriteAPI, failedChan)
			break outputloop
		}
	}
}
