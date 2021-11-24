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
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	allowedNames     = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_:]*$")
	allowedFirstChar = regexp.MustCompile("^[a-zA-Z]")
	allowedFields    = regexp.MustCompile("^[a-zA-Z0-9_:]*$")
	replaceChars     = regexp.MustCompile("[^a-zA-Z0-9_:]")
	allowedTagKeys   = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_]*$")
)

func filterMsg(thread int, msg InfluxMetric, normalize bool) (InfluxMetric, error) {
	FilterTimeStart := time.Now()
	finalMsg := InfluxMetric{
		Name: "", Fields: make(map[string]interface{}),
		Tags: make(map[string]string), Timestamp: msg.Timestamp,
	}
	var name string

	// Step one in filtering: make sure metrics follow prometheus data model (https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels)
	if !allowedFirstChar.MatchString(msg.Name) {
		DroppedMsgs.Inc()
		log.WithFields(log.Fields{"threadNum": thread, "name": msg.Name, "metric": msg, "section": "filter"}).Warning("Dropped! Bad first character in name")
		FilterTime.Add(float64(time.Now().Sub(FilterTimeStart)) / TimeSegmentDivisor)
		return InfluxMetric{}, fmt.Errorf("Dropped metric with bad name")
	}
	// Also check for metrics with no fields (this occasionally happens with histograms from prometheus data sources)
	if len(msg.Fields) < 1 {
		DroppedMsgs.Inc()
		log.WithFields(log.Fields{"threadNum": thread, "name": msg.Name, "metric": msg, "section": "filter"}).Warning("Dropped! No fields in incoming metric")
		FilterTime.Add(float64(time.Now().Sub(FilterTimeStart)) / TimeSegmentDivisor)
		return InfluxMetric{}, fmt.Errorf("Dropped metric with bad name")
	}
	if normalize {
		name = strings.ToLower(msg.Name)
	} else {
		name = msg.Name
	}
	// replace bad characters in metric name with _ per the data model
	if !allowedNames.MatchString(name) {
		FilterSteps.Inc()
		finalMsg.Name = replaceChars.ReplaceAllString(name, "_")
	} else {
		finalMsg.Name = name
	}
	// replace bad characters in tag keys with _ per the data model
	for key, value := range msg.Tags {
		if normalize {
			key = strings.ToLower(key)
			value = strings.ToLower(value)
		}
		if !allowedTagKeys.MatchString(key) {
			FilterSteps.Inc()
			key = replaceChars.ReplaceAllString(key, "_")
		}
		// tags that start with __ are considered custom stats for internal prometheus stuff, we should drop them
		if strings.HasPrefix(key, "__") {
			FilterSteps.Inc()
		} else {
			finalMsg.Tags[key] = value
		}
	}
	// replace bad characters in field names with _ per the data model
	for key, value := range msg.Fields {
		if normalize {
			key = strings.ToLower(key)
		}
		if !allowedFields.MatchString(key) {
			FilterSteps.Inc()
			key = replaceChars.ReplaceAllString(key, "_")
			finalMsg.Fields[key] = value
		} else {
			finalMsg.Fields[key] = value
		}
	}
	MetricsCounted.Add(len(finalMsg.Fields))
	FilterTime.Add(float64(time.Now().Sub(FilterTimeStart)) / TimeSegmentDivisor)
	return finalMsg, nil
}

// FilterMessages is our main loop for ensuring incoming messages match our expected format
func FilterMessages(ctx context.Context, thread int, inChannel chan InfluxMetric, outChannel chan InfluxMetric, wg *sync.WaitGroup, normalize bool) {
	log.WithFields(log.Fields{"threadNum": thread, "section": "filter"}).Info("Starting Message Filtering thread...")
	defer wg.Done()

filterloop:
	for {
		select {
		case msg := <-inChannel:
			output, err := filterMsg(thread, msg, normalize)
			if err == nil {
				outChannel <- output
			}
		case <-ctx.Done():
			log.WithFields(log.Fields{"threadNum": thread, "section": "filter"}).Info("Closing filter thread...")
			for msg := range inChannel {
				output, err := filterMsg(thread, msg, normalize)
				if err == nil {
					outChannel <- output
				}
			}
			break filterloop
		}
	}
}
