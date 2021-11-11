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
from uuid import uuid4
import random
import requests
import time


def write_points(tsd_endpoint, points):
    print("Writing {} points".format(len(points)))
    encoded = "\n".join(points)
    length = len(encoded)
    # encoded = encoded.encode()
    headers = {"Content-Length": str(length), "Content-Type": "text/plain; charset=utf-8"}
    try:
        res = requests.post(tsd_endpoint, data=encoded, headers=headers)
    except Exception as e:
        print("Failed to write points to {} :: {}".format(tsd_endpoint, e))
    else:
        print("Wrote to {} :: {} {}".format(tsd_endpoint, res.status_code, res.text))


def make_points(metric_name, cardinality, write_size):
    additional_tags = "dc=undef,rack=undef,row=undef,group=monitoring"
    # int(time.time()) converts to seconds, but we need nanoseconds...
    ts = time.time()
    timestamp = int(ts * 1000000000)
    print("Using timestamp {} (from {})".format(timestamp, ts))
    metrics = []
    for count in range(cardinality):
        metrics.append("{},host={},{} value={} {}".format(metric_name, uuid4(), additional_tags,
                                                          random.random() * 1000, timestamp))
        if count % write_size == 0:
            yield metrics
            metrics = []
    yield metrics


def cli_opts():
    parser = ArgumentParser(description="Produce stat with high cardinality",
                            formatter_class=ArgumentDefaultsHelpFormatter)
    parser.add_argument("--debug", action="store_true", default=False,
                        help="Show debug information")
    parser.add_argument("--metric-name", default="test_high_card_metric", help="Our metric name")
    parser.add_argument("-c", "--cardinality", type=int, default=10000, help="Cardinality size")
    parser.add_argument("-w", "--write-size", type=int, default=1000, help="Write chunk size")
    parser.add_argument("--tsd-endpoint",
            help="endpoint to write to (currently in influx format)")
    return parser.parse_args()


def main():
    opts = cli_opts()
    for points in make_points(opts.metric_name, opts.cardinality, opts.write_size):
        write_points(opts.tsd_endpoint, points)


if __name__ == "__main__":
    main()
