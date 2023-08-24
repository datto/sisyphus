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
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pkg/profile"
	log "github.com/sirupsen/logrus"
)

/*
Pipeline :
meta struct to track our pipelines across threads
*/
type Pipeline struct {
	/*
		Pipeline meta variables
		(Context and waitgroups)
		Allows for signal management of
		each pipeline across threads
	*/
	Ctx          context.Context
	ReadCTX      context.Context
	ReadCancel   context.CancelFunc
	JSONCTX      context.Context
	JSONCancel   context.CancelFunc
	FilterCTX    context.Context
	FilterCancel context.CancelFunc
	OutputCTX    context.Context
	OutputCancel context.CancelFunc
	FailedCTX    context.Context
	FailedCancel context.CancelFunc
	ReadWG       sync.WaitGroup
	JSONWG       sync.WaitGroup
	FilterWG     sync.WaitGroup
	WriteWG      sync.WaitGroup
	FailedWG     sync.WaitGroup
	/*
		Actual variables needed for processing
		data in the pipeline
	*/
	TSDURL                string
	ProcessInfluxJSONChan chan []byte
	ProcessInfluxLineChan chan []byte
	ProcessPromJSONChan   chan []byte
	FilterTagChan         chan InfluxMetric
	OutputTSDBChan        chan InfluxMetric
	FailedWritesChan      chan string
}

var (
	//Endpoints is our meta object to track each pipeline
	Endpoints []Pipeline
	//Version : part of our version string
	Version = "development"
	//BuildTime : part of our version string
	BuildTime = "now"
	//BuildUser : part of our version string
	BuildUser = "none"
)

func main() {
	var err error
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	var cfgfile = flag.String("config", "config.yml", "Full path to config file")
	var memprofile = flag.Bool("memprofile", false, "Enable memory profiling")
	var cpuprofile = flag.Bool("cpuprofile", false, "Enable CPU profiling")
	var versioninfo = flag.Bool("version", false, "Display Version Info and exit")
	var debug = flag.Bool("debug", false, "Debug logging")
	flag.Parse()
	if *versioninfo {
		fmt.Println("Version:\t", Version)
		fmt.Println("Build Time:\t", BuildTime)
		fmt.Println("Build User:\t", BuildUser)
		os.Exit(0)
	}
	if *debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	if *cpuprofile {
		defer profile.Start().Stop()
	}
	if *memprofile {
		defer profile.Start(profile.MemProfile).Stop()
	}
	log.WithFields(log.Fields{"Version": Version}).Info("Sisyphus starting...")
	var c Config
	c.LoadConfig(*cfgfile)
	if err != nil {
		panic(err)
	}
	/*
		Shouldn't allow for many signals to buffer
		Process each one before moving forward
	*/
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	log.WithFields(log.Fields{"Configs": c.WritePaths, "Length": len(c.WritePaths)}).Debug("Creating endpoint structs")
	Endpoints := make([]Pipeline, len(c.WritePaths))
	go StatsListener(c.StatsAddress, c.StatsPort)

	for i := 0; i < len(c.WritePaths); i++ {
		/*
			Properly format write endpoint
		*/
		if c.WritePaths[i].TSDPort != "" {
			Endpoints[i].TSDURL = fmt.Sprintf("%v:%v", c.WritePaths[i].TSDEndpoint, c.WritePaths[i].TSDPort)
		} else {
			Endpoints[i].TSDURL = fmt.Sprintf("%v", c.WritePaths[i].TSDEndpoint)
		}
		Endpoints[i].TSDURL = fmt.Sprintf("%v%v", Endpoints[i].TSDURL, c.WritePaths[i].TSDURLPath)
		log.WithFields(log.Fields{"TSDURL": Endpoints[i].TSDURL, "section": "main"}).Info("Output URL")
		/*
			Build meta objects
			(contexts and wait groups)
		*/
		Endpoints[i].Ctx = context.Background()
		Endpoints[i].ReadCTX, Endpoints[i].ReadCancel = context.WithCancel(Endpoints[i].Ctx)
		Endpoints[i].JSONCTX, Endpoints[i].JSONCancel = context.WithCancel(Endpoints[i].Ctx)
		Endpoints[i].FilterCTX, Endpoints[i].FilterCancel = context.WithCancel(Endpoints[i].Ctx)
		Endpoints[i].OutputCTX, Endpoints[i].OutputCancel = context.WithCancel(Endpoints[i].Ctx)
		Endpoints[i].FailedCTX, Endpoints[i].FailedCancel = context.WithCancel(Endpoints[i].Ctx)

		/*
			Channels

			Actually create all the channels with the defined
			buffer size
		*/
		Endpoints[i].ProcessInfluxJSONChan = make(chan []byte, c.WritePaths[i].ChannelSize)
		Endpoints[i].ProcessInfluxLineChan = make(chan []byte, c.WritePaths[i].ChannelSize)
		Endpoints[i].ProcessPromJSONChan = make(chan []byte, c.WritePaths[i].ChannelSize)
		Endpoints[i].FilterTagChan = make(chan InfluxMetric, c.WritePaths[i].ChannelSize)
		Endpoints[i].OutputTSDBChan = make(chan InfluxMetric, c.WritePaths[i].ChannelSize)
		Endpoints[i].FailedWritesChan = make(chan string, c.WritePaths[i].ChannelSize)

		/*
			Go Routines

			First routine is a single thread for failed writes
			This shouldn't need more than one thread because
			it should be low-volume
		*/
		Endpoints[i].FailedWG.Add(1)
		go SendFailedToKafka(Endpoints[i].FailedCTX, Endpoints[i].FailedWritesChan, KafkaProducerMeta{Topic: c.FailedWritesTopic,
			Brokers: c.BrokerStr, CompressionType: c.FailedWritesCompression,
			WritePath: Endpoints[i].TSDURL, TSDOrg: c.WritePaths[i].TSDDBOrg,
			TSDName: c.WritePaths[i].TSDDBName}, &Endpoints[i].FailedWG)
		/*
			Next we add processing threads
			This is the step that consumes from Kafka
			and deserializes messages. We'll want *at least*
			one of these for each topic, and likely many
		*/
		for thread := 1; thread <= c.WritePaths[i].ProcessThreads; thread++ {
			if len(c.WritePaths[i].InfluxJSONTopics) > 0 {
				Endpoints[i].JSONWG.Add(1)
				go ProcessInfluxJSONMsg(Endpoints[i].JSONCTX, thread, Endpoints[i].ProcessInfluxJSONChan, Endpoints[i].FilterTagChan, &Endpoints[i].JSONWG, c.WritePaths[i].FlipSingleFields)
			}
			if len(c.WritePaths[i].PromTopics) > 0 {
				Endpoints[i].JSONWG.Add(1)
				go ProcessPromMsg(Endpoints[i].JSONCTX, thread, Endpoints[i].ProcessPromJSONChan, Endpoints[i].OutputTSDBChan, c.Normalize, c.WritePaths[i].FlipSingleFields, &Endpoints[i].JSONWG)
			}
			if len(c.WritePaths[i].InfluxLineTopics) > 0 {
				Endpoints[i].JSONWG.Add(1)
				go ProcessInfluxLineMsg(Endpoints[i].JSONCTX, thread, Endpoints[i].ProcessInfluxLineChan, Endpoints[i].FilterTagChan, &Endpoints[i].JSONWG, c.WritePaths[i].FlipSingleFields)
			}
		}
		/*
			We're only going to filter influx-style metrics
			because they're a more generally permissive format,
			so there's more things to filter.
			Prometheus metrics should already meet the Prometheus data model...
		*/
		for thread := 1; thread <= c.WritePaths[i].FilterThreads; thread++ {
			Endpoints[i].FilterWG.Add(1)
			go FilterMessages(Endpoints[i].FilterCTX, thread, Endpoints[i].FilterTagChan, Endpoints[i].OutputTSDBChan, &Endpoints[i].FilterWG, c.Normalize)
		}
		/*
			Output threads...
			As above, we define as many as requested per write path
		*/
		for thread := 1; thread <= c.WritePaths[i].WriteThreads; thread++ {
			Endpoints[i].WriteWG.Add(1)
			cfg := OutputMeta{Thread: thread, BatchSize: c.WritePaths[i].SendBatch, WriteTimeout: c.WritePaths[i].WriteTimeout,
				MaxRetries: c.WritePaths[i].MaxRetries, FlushSegment: c.WritePaths[i].TSDFlushSegment, URL: Endpoints[i].TSDURL,
				TsdOrg: c.WritePaths[i].TSDDBOrg, TsdDbName: c.WritePaths[i].TSDDBName}
			go SendTSDB(Endpoints[i].OutputCTX, Endpoints[i].OutputTSDBChan, Endpoints[i].FailedWritesChan, cfg, &Endpoints[i].WriteWG)
		}
		/*
			Actual kafka threads, connected to the Process threads
			We initialize these last to have the rest of the pipeline
			ready before we start actually consuming data
		*/
		for thread := 1; thread <= c.WritePaths[i].ReadThreads; thread++ {
			if len(c.WritePaths[i].InfluxJSONTopics) > 0 {
				Endpoints[i].ReadWG.Add(1)
				cfg := KafkaConsumerMeta{ThreadCount: thread, Topics: c.WritePaths[i].InfluxJSONTopics,
					Brokers: c.BrokerStr, ConsumerGroup: c.ConsumerGroup,
					ClientID: c.ClientID, SessionTimeout: c.SessionTimeout,
					OffsetReset: c.Offset}
				go ReadFromKafka(Endpoints[i].ReadCTX, cfg, Endpoints[i].ProcessInfluxJSONChan, &Endpoints[i].ReadWG)
			}
			if len(c.WritePaths[i].InfluxLineTopics) > 0 {
				Endpoints[i].ReadWG.Add(1)
				cfg := KafkaConsumerMeta{ThreadCount: thread, Topics: c.WritePaths[i].InfluxLineTopics,
					Brokers: c.BrokerStr, ConsumerGroup: c.ConsumerGroup,
					ClientID: c.ClientID, SessionTimeout: c.SessionTimeout,
					OffsetReset: c.Offset}
				go ReadFromKafka(Endpoints[i].ReadCTX, cfg, Endpoints[i].ProcessInfluxJSONChan, &Endpoints[i].ReadWG)
			}
			if len(c.WritePaths[i].PromTopics) > 0 {
				Endpoints[i].ReadWG.Add(1)
				cfg := KafkaConsumerMeta{ThreadCount: thread, Topics: c.WritePaths[i].PromTopics,
					Brokers: c.BrokerStr, ConsumerGroup: c.ConsumerGroup,
					ClientID: c.ClientID, SessionTimeout: c.SessionTimeout,
					OffsetReset: c.Offset}
				go ReadFromKafka(Endpoints[i].ReadCTX, cfg, Endpoints[i].ProcessPromJSONChan, &Endpoints[i].ReadWG)
			}
		}
	}

	run := true
	for run {
		/*
			Shut down steps...
			Because we have wait groups stored for each pipeline object,
			we can just loop through the WritePaths object and issue
			cancel/wait commands to the wait groups and move along
		*/
		sig := <-sigchan
		log.WithFields(log.Fields{"error": sig, "section": "main"}).Error("Caught signal...terminating")
		for i := 0; i < len(c.WritePaths); i++ {
			log.WithFields(log.Fields{"queue": i}).Info("Closing ingest threads for writepath")
			Endpoints[i].ReadCancel()
			Endpoints[i].ReadWG.Wait()
			log.WithFields(log.Fields{"Influx Proccess Queue": len(Endpoints[i].ProcessInfluxJSONChan), "Prometheus Process Queue": len(Endpoints[i].ProcessPromJSONChan), "section": "main"}).Info("Waiting on queues to flush...")
			close(Endpoints[i].ProcessInfluxJSONChan)
			close(Endpoints[i].ProcessPromJSONChan)
			Endpoints[i].JSONCancel()
			Endpoints[i].JSONWG.Wait()
			log.WithFields(log.Fields{"Filter Queue": len(Endpoints[i].FilterTagChan), "section": "main"}).Info("Waiting on queues to flush...")
			close(Endpoints[i].FilterTagChan)
			Endpoints[i].FilterCancel()
			Endpoints[i].FilterWG.Wait()
			log.WithFields(log.Fields{"Output Queue": len(Endpoints[i].OutputTSDBChan), "section": "main"}).Info("Waiting on queues to flush...")
			close(Endpoints[i].OutputTSDBChan)
			Endpoints[i].OutputCancel()
			Endpoints[i].WriteWG.Wait()
			log.WithFields(log.Fields{"Failed Write Queue": len(Endpoints[i].FailedWritesChan), "section": "main"}).Info("Waiting on queues to flush...")
			close(Endpoints[i].FailedWritesChan)
			Endpoints[i].FailedCancel()
			Endpoints[i].FailedWG.Wait()
		}
		log.WithFields(log.Fields{"section": "main"}).Info("Queues flushed. Exiting.")
		run = false
	}
}
