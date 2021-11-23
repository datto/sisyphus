# sisyphus

Kafka -> Influx 2.x forwarder in Golang

# Expectations

1. Incoming data is formatted as:
  * "Influx JSON" format (telegraf's JSON output: https://github.com/influxdata/telegraf/tree/master/plugins/serializers/json)
  * Influx line protocol (https://github.com/influxdata/telegraf/tree/master/plugins/serializers/influx)
  * "Prometheus JSON" format (created from https://github.com/Telefonica/prometheus-kafka-adapter#json)
2. Outbound data writes to an Influx v2 compatible endpoint.

# Guarantees

* Output data is formatted in an prometheus-compatible format (https://prometheus.io/docs/concepts/data_model/)
* Failed writes are written to a defined Kafka topic for later inspection/processing

# Config
```
consumer_group: sisyphus-test
client_id: <hostname>

brokers:
  - localhost:9092
starting_offset_type: earliest
normalize_metrics: false

writepaths:
  - influx_json_topics:
      - test
      - test2
  - influx_line_topics:
      - test5
      - test6
    prometheus_topics:
      - test3
      - test4
    output_hostname: localhost
    output_path: "/insert/0:0/influx"
    output_port: 8480
    go_channel_size: 10000
    # number of points to send on each write
    send_batch: 1000
    # in seconds
    tsd_flush_time: 10
    # milliseconds before a write is considered timed out
    write_timeout: 30
    # individual thread sizes
    kafka_reader_threads: 1
    processor_threads: 1
    filter_threads: 1
    write_threads: 1
    flip_single_fields: true
    max_retries: 0

failed_writes_topic: influx-failed-writes

stats_listen_address: 127.0.0.1
stats_listen_port: 9999
```

## `normalize_metrics`

Forces all incoming stat names/tags/field names to lowercase strings. This is useful if you are trying to migrate a from a system that already did this.

## `flip_single_fields`

This is a strange config, but needed when using VictoriaMetrics as a destination with `-influxSkipSingleField`. You can end up needed this in the following situation:

1. You are ingesting data from influx sources
2. you would like to add prometheus sources

To successfully change the data model, we must enable `-influxSkipSingleField` so prometheus stats like:
```
{"value": 2,
 "timestamp": 0,
 "labels": {
   "__name__": "up",
   "dc": "dc1"
 }
}
```
Will not produce a value like `up_value` in VictoriaMetrics.

But this means that an influx stat like:
```
{"fields": {
   "lag": 2
 },
 "tags": {
   "dc": "dc1"
 },
 "name": "kafka",
 "timestamp": 0
}
```
Will become `kafka` (dropping the `_lag` value that is descriptive and useful).

In this case, we can enable `flip_single_fields`, and `kafka_lag` will be submitted as `kafka_lag_value`, which VictoriaMetrics will then trim to `kafka_lag`.

# Stats

Sisyphus uses https://github.com/VictoriaMetrics/metrics to produce Prometheus compatible stats

Stats available `http://<stats_listen_address>:<stats_listen_port>/<stats_listen_path>`.

A quick breakdown:

* ReceivedMsgs
  * Messages that have been received and marshalled into objects for processing
* SentMsgs
  * Messages sent to our output
* FilteredMsgs
  * Messages that have been altered in some way to conform to standards (removed tag, changed key name, etc.)
* DroppedMsgs
  * Messages that can't be fixed to meet standards and must be dropped
* FailedMsgs
  * Messages that failed to send to the output
  * failed messages are added to a dead-letter queue in Kafka
* IngestMsgs
  * Messages initially received from Kafka
* MetricsCounted
  * Individual metrics counted during the filtering process

# Licensing

sisyphus is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, under version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
