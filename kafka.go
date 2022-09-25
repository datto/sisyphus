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
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	json "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

/*
KafkaConsumerMeta objects for creating Kafka functions
(makes for cleaner passing of data)
*/
type KafkaConsumerMeta struct {
	ThreadCount    int
	Topics         []string
	Brokers        string
	ConsumerGroup  string
	ClientID       string
	SessionTimeout int
	OffsetReset    string
}

// KafkaProducerMeta : meta about Kafka producer objects
type KafkaProducerMeta struct {
	Topic           string
	Brokers         string
	CompressionType string
	WritePath       string
	TSDOrg          string
	TSDName         string
}

/*
DeadLetterMsg :
Each message to the dead letter queue may be for a different tenant
We'll need this struct to ensure we have metadata
*/
type DeadLetterMsg struct {
	WritePath string
	TSDOrg    string
	TSDName   string
	Message   string
}

var (
	deliveryChan chan kafka.Event
)

/*
Dead letter queue section

Here we collect messages that didn't write successfully and push them back into Kafka
*/
func processFailed(msg DeadLetterMsg, topic string, producer *kafka.Producer) {
	FailedTimeStart := time.Now()
	thisMsg, err := json.Marshal(msg)
	if err != nil {
		log.WithFields(log.Fields{"message": msg, "error": err}).Fatal("Failed to deserialize message for dead letter queue")
	}
	err = producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          thisMsg,
	}, nil)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Fatal("Couldn't write message to dead letter queue")
	}
	FailedMsgs.Inc()
	FailedWriteTime.Add(float64(time.Now().Sub(FailedTimeStart)) / TimeSegmentDivisor)
}

// SendFailedToKafka : Exposed function for sending failed write attempts to our dead letter queue
func SendFailedToKafka(ctx context.Context, channel chan string, prodMeta KafkaProducerMeta, wg *sync.WaitGroup) {
	/*
		Dead letter messages should contain:
		1. the endpoint they were being sent to (to handle tenancy)
		2. the actual metric that failed to write
		3. the topic they came from
	*/
	log.WithFields(log.Fields{"section": "failedwrites"}).Info("Starting failed writes thread...")
	defer wg.Done()
	producer, err := kafka.NewProducer(
		&kafka.ConfigMap{
			"bootstrap.servers":   prodMeta.Brokers,
			"compression.type":    prodMeta.CompressionType,
			"go.delivery.reports": false})
	if err != nil {
		log.WithFields(log.Fields{"error": err, "section": "failedwrites"}).Fatal("Couldn't build Kafka producer")
	}
	defer producer.Close()

failedloop:
	for {
		select {
		case msg := <-channel:
			processFailed(DeadLetterMsg{Message: msg, WritePath: prodMeta.WritePath,
				TSDOrg: prodMeta.TSDOrg, TSDName: prodMeta.TSDName}, prodMeta.Topic, producer)
		case <-ctx.Done():
			// we want to drain the queue before we completely close (if possible)
			log.WithFields(log.Fields{"section": "failedwrites"}).Info("Closing failed writes thread...")
			for msg := range channel {
				processFailed(DeadLetterMsg{Message: msg, WritePath: prodMeta.WritePath,
					TSDOrg: prodMeta.TSDOrg, TSDName: prodMeta.TSDName}, prodMeta.Topic, producer)
			}
			// wait for producer to flush for 10 seconds
			producer.Flush(10 * 1000)
			break failedloop
		}
	}
}

// ReadFromKafka : Allow for reading Influx or Prometheus-style stats through a boolean
func ReadFromKafka(ctx context.Context, cfg KafkaConsumerMeta, outputChannel chan []byte, wg *sync.WaitGroup) {
	log.WithFields(log.Fields{"threadNum": cfg.ThreadCount, "section": "kafka reader"}).Info("Starting Sisyphus ingest thread...")
	defer wg.Done()
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        cfg.Brokers,
		"client.id":                fmt.Sprintf("%v-%v", cfg.ClientID, cfg.ThreadCount),
		"group.id":                 cfg.ConsumerGroup,
		"session.timeout.ms":       cfg.SessionTimeout,
		"go.events.channel.enable": true,
		"enable.partition.eof":     true,
		"enable.auto.commit":       true,
		"auto.offset.reset":        cfg.OffsetReset})
	if err != nil {
		log.WithFields(log.Fields{"threadNum": cfg.ThreadCount, "error": err, "section": "kafka reader"}).Fatal("Couldn't build consumer")
	}
	defer consumer.Close()
	err = consumer.SubscribeTopics(cfg.Topics, nil)
	if err != nil {
		log.WithFields(log.Fields{"threadNum": cfg.ThreadCount, "error": err, "section": "kafka reader"}).Fatal("Couldn't subscribe to topics")
	}
	log.WithFields(log.Fields{"threadNum": cfg.ThreadCount, "brokers": cfg.Brokers, "topics": cfg.Topics, "section": "kafka reader"}).Info("Consumer Started")

readloop:
	for {
		select {
		case ev := <-consumer.Events():
			ingestTimeStart := time.Now()
			switch e := ev.(type) {
			case kafka.AssignedPartitions:
				log.WithFields(log.Fields{"threadNum": cfg.ThreadCount, "msg": e, "section": "kafka reader"}).Debug("Assigning partitions...")
				err := consumer.Assign(e.Partitions)
				if err != nil {
					log.WithFields(log.Fields{"threadNum": cfg.ThreadCount, "msg": e, "section": "kafka reader"}).Error("Couldn't assign partitions")
				}
			case kafka.RevokedPartitions:
				log.WithFields(log.Fields{"threadNum": cfg.ThreadCount, "msg": e, "section": "kafka reader"}).Debug("Revoked partitions...")
				err := consumer.Unassign()
				if err != nil {
					log.WithFields(log.Fields{"threadNum": cfg.ThreadCount, "msg": e, "section": "kafka reader"}).Error("Couldn't unassign partitions")
				}
			case kafka.PartitionEOF:
				log.WithFields(log.Fields{"threadNum": cfg.ThreadCount, "msg": e, "section": "kafka reader"}).Debug("End of partition...")
			case *kafka.Message:
				IngestMsgs.Inc()
				outputChannel <- e.Value
			case kafka.Error:
				// Errors should generally be considered as informational, the client will try to automatically recover
				log.WithFields(log.Fields{"threadNum": cfg.ThreadCount, "error": e, "section": "kafka reader"}).Error("Kafka Error, recovering...")
			}
			IngestTime.Add(float64(time.Now().Sub(ingestTimeStart)) / TimeSegmentDivisor)
		case <-ctx.Done():
			log.WithFields(log.Fields{"threadNum": cfg.ThreadCount, "section": "kafka reader"}).Info("Closing ingest thread...")
			break readloop
		}
	}
}
