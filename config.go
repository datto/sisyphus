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
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	// ResponseTimeout sets a default for how long HTTP responses should take
	ResponseTimeout = 30
	// DefaultChannelSize sets a default for how large each go channel should be
	DefaultChannelSize = 10000
	// DefaultOffset defines the basic kafka offset behavior for new consumer groups
	DefaultOffset = "earliest"
	// DefaultSendBatch defines the length of our metric array we should send to outputs
	DefaultSendBatch = 1000
	// DefaultSessionTimeout defines how frequently we should attempt to re-negotiate HTTP connections
	DefaultSessionTimeout = 6000
	// DefaultStatsAddress defines the listener address for our prometheus stats
	DefaultStatsAddress = "127.0.0.1"
	// DefaultStatsPort defines the listener port for our prometheus stats
	DefaultStatsPort = "9999"
	// DefaultTSDFlushSegment defines how frequently we should force writes to outputs in seconds
	DefaultTSDFlushSegment = 5
	// TimeSegmentDivisor defines how we should segment time-based decisions (currently in seconds (nanosecond * microsecond * millisecond * second))
	TimeSegmentDivisor = 1000 * 1000 * 1000 * 1
)

var (
	// DefaultThreads sits at half the CPU cores on the host, we can tune this if needed
	DefaultThreads = int(runtime.NumCPU() / 2)
	// Currently we'll stick with the old default of 1 thread
	// DefaultThreads = 1
)

// PromMetric defines a single Prometheus metric as an in-memory object
type PromMetric struct {
	Value     string            `json:"value"`
	Labels    map[string]string `json:"labels"`
	Name      string            `json:"name"`
	Timestamp string            `json:"timestamp"`
}

// InfluxMetric defines a single Influx metric as an in-memory object
type InfluxMetric struct {
	Fields    map[string]interface{} `json:"fields"`
	Tags      map[string]string      `json:"tags"`
	Name      string                 `json:"name"`
	Timestamp int64                  `json:"timestamp"`
}

// WritePath holds metadata about an output path
type WritePath struct {
	TSDEndpoint      string   `yaml:"output_endpoint"`
	TSDURLPath       string   `yaml:"output_path"`
	TSDPort          string   `yaml:"output_port"`
	TSDDBName        string   `yaml:"tsd_database_name"`
	TSDDBOrg         string   `yaml:"tsd_database_org"`
	PromTopics       []string `yaml:"prometheus_topics"`
	InfluxJSONTopics []string `yaml:"influx_json_topics"`
	InfluxLineTopics []string `yaml:"influx_line_topics"`

	// threading settings
	ChannelSize    int `yaml:"go_channel_size"`
	ReadThreads    int `yaml:"kafka_reader_threads"`
	ProcessThreads int `yaml:"processor_threads"`
	FilterThreads  int `yaml:"filter_threads"`
	WriteThreads   int `yaml:"write_threads"`

	// output tuning
	SendBatch       uint    `yaml:"send_batch"`
	WriteTimeout    uint    `yaml:"write_timeout"`
	TSDFlushSegment float64 `yaml:"tsd_flush_time"`
	MaxRetries      uint    `yaml:"max_retries"`

	// misc
	FlipSingleFields bool `yaml:"flip_single_fields"`
}

// Config holds general config data (and is the "top level" of the config object we load from our config yaml file)
type Config struct {
	ConsumerGroup           string   `yaml:"consumer_group"`
	ClientID                string   `yaml:"client_id"`
	Brokers                 []string `yaml:"brokers"`
	SessionTimeout          int      `yaml:"kafka_session_timeout"`
	BrokerStr               string
	FailedWritesTopic       string `yaml:"failed_writes_topic"`
	FailedWritesCompression string `yaml:"failed_writes_compression_type"`
	Offset                  string `yaml:"starting_offset_type"`
	Normalize               bool   `yaml:"normalize_metrics"`

	TLSCA   string `yaml:"tls_ca"`
	TLSCert string `yaml:"tls_cert"`
	TLSKey  string `yaml:"tls_key"`

	WritePaths []WritePath

	StatsAddress string `yaml:"stats_listen_address"`
	StatsPort    string `yaml:"stats_listen_port"`
}

// LoadConfig actually loads our config file and sets some defaults
func (c *Config) LoadConfig(ConfigFile string) {
	log.Info("Loading config...")
	var err error

	filename, err := filepath.Abs(ConfigFile)
	if err != nil {
		panic(err)
	}
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		panic(err)
	}

	/*
		Set defaults for individual write paths
	*/
	for i := 0; i < len(c.WritePaths); i++ {
		for index, element := range c.WritePaths[i].PromTopics {
			if !strings.ContainsAny(element, "^*$") {
				c.WritePaths[i].PromTopics[index] = fmt.Sprintf("^%v$", element)
			}
		}
		for index, element := range c.WritePaths[i].InfluxJSONTopics {
			if !strings.ContainsAny(element, "^*$") {
				c.WritePaths[i].InfluxJSONTopics[index] = fmt.Sprintf("^%v$", element)
			}
		}
		for index, element := range c.WritePaths[i].InfluxLineTopics {
			if !strings.ContainsAny(element, "^*$") {
				c.WritePaths[i].InfluxLineTopics[index] = fmt.Sprintf("^%v$", element)
			}
		}
		if c.WritePaths[i].TSDEndpoint == "" {
			c.WritePaths[i].TSDEndpoint = "http://localhost"
		}
		if c.WritePaths[i].TSDURLPath == "" {
			c.WritePaths[i].TSDURLPath = "/"
		}
		/*
			Set defaults for threading and channel sizes
		*/
		if c.WritePaths[i].ReadThreads == 0 {
			c.WritePaths[i].ReadThreads = DefaultThreads
		}
		if c.WritePaths[i].ProcessThreads == 0 {
			c.WritePaths[i].ProcessThreads = DefaultThreads
		}
		if c.WritePaths[i].FilterThreads == 0 {
			c.WritePaths[i].FilterThreads = DefaultThreads
		}
		if c.WritePaths[i].WriteThreads == 0 {
			c.WritePaths[i].WriteThreads = DefaultThreads
		}
		if c.WritePaths[i].ChannelSize == 0 {
			c.WritePaths[i].ChannelSize = DefaultChannelSize
		}
		/*
			Set defaults for write configs
		*/
		if c.WritePaths[i].SendBatch == 0 {
			c.WritePaths[i].SendBatch = DefaultSendBatch
		}
		if c.WritePaths[i].WriteTimeout == 0 {
			c.WritePaths[i].WriteTimeout = ResponseTimeout
		}
		if c.WritePaths[i].TSDFlushSegment == 0 {
			c.WritePaths[i].TSDFlushSegment = DefaultTSDFlushSegment
		}
	}
	if c.FailedWritesCompression == "" {
		c.FailedWritesCompression = "gzip"
	}
	if c.ClientID == "" {
		c.ClientID, err = os.Hostname()
		if err != nil {
			panic(err)
		}
	}
	if c.SessionTimeout == 0 {
		c.SessionTimeout = DefaultSessionTimeout
	}
	if c.Brokers == nil {
		c.BrokerStr = "localhost:9092"
	} else {
		c.BrokerStr = strings.Join(c.Brokers, ",")
	}
	if c.Offset == "" {
		c.Offset = DefaultOffset
	}

	/*
		Set defaults for stats configs
	*/
	if c.StatsAddress == "" {
		c.StatsAddress = DefaultStatsAddress
	}
	if c.StatsPort == "" {
		c.StatsPort = DefaultStatsPort
	}
	log.WithFields(log.Fields{"config": c}).Debug("Config!")
}
