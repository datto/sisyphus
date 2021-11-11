"""
 * This file is part of sisyphus.
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
 *
"""
#!/usr/bin/env python3

from argparse import ArgumentParser, ArgumentDefaultsHelpFormatter
from confluent_kafka import Consumer, KafkaError
import json
import logging
from pprint import pformat
import requests
from uuid import uuid4

log = logging.getLogger("failed-message-collector")
log.setLevel(logging.INFO)
ch = logging.StreamHandler()
logformat = '%(asctime)s %(name)s %(levelname)s %(message)s'
formatter = logging.Formatter(logformat)
ch.setFormatter(formatter)
log.addHandler(ch)


class KafkaToInflux(object):
    def __init__(self, config):
        self.batch_size = config["batch_size"]
        self.chunk_size = config["chunk_size"]
        self.normalize = config["normalize"]
        self.c = Consumer({
            'bootstrap.servers': config["brokers"], 'group.id': config["group"],
            'client.id': uuid4(), 'enable.auto.commit': True,
            'auto.offset.reset': 'earliest'
        })
        self.c.subscribe([config["topic"]])
        self.timeout = config["timeout"]
        self.write_postfix = "api/v2/write?precision={}".format(config["precision"])

    def read_msgs(self):
        msgs = []
        msg_count = 0
        output_path = ""
        tsdb_name = ""
        tsdb_org = ""
        while True:
            msg = self.c.poll(timeout=self.timeout)
            if msg is None:
                log.debug("No messages in {} seconds...".format(self.timeout))
                break
            if msg.error():
                if msg.error().code() == KafkaError._PARTITION_EOF:
                    continue
                else:
                    log.fatal("Actual kafka error: {}".format(msg.error().str()))
                    raise
            try:
                payload = msg.value().decode('utf-8').strip("\n")
            except Exception as e:
                log.warning("Failed to load {} :: {}".format(msg.value(), e))
                continue
            try:
                payload = json.loads(payload)
            except Exception as e:
                log.warning("Failed to deserialize {} :: {}".format(payload, e))
                continue
            log.debug("Found a message: {}".format(pformat(payload)))
            if not payload or not payload.get("WritePath", None) or payload["Message"].startswith(","):
                log.warning("{} doesn't have a proper name".format(payload))
                continue
            msg_count += 1
            if self.normalize:
                payload["Message"] = payload["Message"].lower()
            if not output_path or payload["WritePath"] != output_path or tsdb_name != payload["TSDName"] or tsdb_org != payload["TSDOrg"]:
                if msgs:
                    yield {"output": output_path, "msgs": msgs, "name": tsdb_name,
                           "org": tsdb_org}
                    msgs = []
                output_path = str(payload["WritePath"])
                tsdb_name = str(payload["TSDOrg"])
                tsdb_org = str(payload["TSDName"])
            msgs.append(payload["Message"].strip("\n"))
            if msg_count >= self.chunk_size:
                yield {"output": output_path, "msgs": msgs, "name": tsdb_name,
                       "org": tsdb_org}
                msgs = []
                log.debug("collected a chunk...")
                break
            if len(msgs) >= self.batch_size:
                yield {"output": output_path, "msgs": msgs, "name": tsdb_name,
                       "org": tsdb_org}
                msgs = []
            if msg_count > self.batch_size and msg_count % self.batch_size == 0:
                log.info("Metrics collected: {}".format(msg_count))
        log.info("Collected {} messages".format(msg_count))
        log.debug("Messages: {}".format(msgs))
        yield {"output": output_path, "msgs": msgs, "name": tsdb_name,
               "org": tsdb_org}

    def write_msgs(self, payload):
        if not payload["msgs"]:
            return
        output = "{}/{}".format(payload["output"], self.write_postfix)
        if payload["org"]:
            output += "&org={}".format(payload["org"])
        if payload["name"]:
            output += "&bucket={}".format(payload["name"])
        log.debug("Writing to {}: batch -> {}".format(output, payload["msgs"]))
        res = requests.post(output,
                            data="\n".join(payload["msgs"]))
        log.debug("Results: {}".format(res.__dict__))


def metric_poll(args):
    log.info("Starting metric polling...")
    config = {"brokers": args["brokers"],
              "group": args["group"],
              "topic": args["topic"],
              "timeout": args.get("timeout", 10),
              "normalize": args.get("normalize", False),
              "precision": args.get("precision", "u"),
              "chunk_size": args.get("chunk_size", 1000000),
              "batch_size": args.get("batch_size", 250)}
    influx = KafkaToInflux(config)
    for msgbatch in influx.read_msgs():
        influx.write_msgs(msgbatch)


def cli_opts():
    parser = ArgumentParser(description="Collect failed writes from Kafka and forward to influx-compat endpoint",
                            formatter_class=ArgumentDefaultsHelpFormatter)
    parser.add_argument("--brokers", type=list, help="Broker list to connect to")
    parser.add_argument("-t", "--topic", help="Topic to read from")
    parser.add_argument("-g", "--group", help="Kafka consumer group name")
    parser.add_argument("-b", "--batch-size", type=int, default=250, help="Size of JSON batches to write to endpoint")
    parser.add_argument("-c", "--chunk-size", type=int, default=20000000, help="Upper limit on messages to read in a run")
    parser.add_argument("--timeout", type=int, default=30, help="Query timeout")
    parser.add_argument("--precision", default="u", help="Precision for timestamps (default is for victoriametrics compatibility)")
    parser.add_argument("--normalize", default=False, action="store_true", help="Normalize metrics when re-writing")
    parser.add_argument("--debug", default=False, action="store_true", help="Debug logging")
    return parser


def main():
    args = cli_opts().parse_args()
    if args.chunk_size < args.batch_size:
        log.warning("Batch collection size smaller than write size!")
        args.chunk_size = args.batch_size
    if args.debug:
        log.setLevel(logging.DEBUG)
    metric_poll(args.__dict__)


if __name__ == '__main__':
    main()
