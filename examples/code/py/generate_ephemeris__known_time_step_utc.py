#  MIT License
#
#  Copyright (c) 2026 Roman Bielyi
#
#  Permission is hereby granted, free of charge, to any person obtaining a copy
#  of this software and associated documentation files (the "Software"), to deal
#  in the Software without restriction, including without limitation the rights
#  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
#  copies of the Software, and to permit persons to whom the Software is
#  furnished to do so, subject to the following conditions:
#
#  The above copyright notice and this permission notice shall be included in all
#  copies or substantial portions of the Software.
#
#  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
#  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
#  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
#  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
#  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
#  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
#  SOFTWARE.

import grpc
import logging
import time
from datetime import datetime, timezone
from helper import get_tls_config

from google.protobuf.timestamp_pb2 import Timestamp
from api.v1.core import ephem_pb2
from api.v1 import common_pb2
from api.v1 import main_pb2_grpc


def main():
    grpc_creds = get_tls_config()

    options = (
        ("grpc.ssl_target_name_override", "xpropagator-server"),
    )

    try:
        with grpc.secure_channel("localhost:50051", grpc_creds, options) as channel:
            client = main_pb2_grpc.PropagatorStub(channel)

            time_start = time.time()

            def make_ts(y, m, d):
                ts = Timestamp()
                ts.FromDatetime(datetime(y, m, d, 0, 0, 0, tzinfo=timezone.utc))
                return ts

            ephem_req = ephem_pb2.EphemRequest(
                req_id=1,
                ephem_type=ephem_pb2.EphemType.EphemJ2K,
                common_time_grid=ephem_pb2.EphemTimeGrid(
                    time_start_utc=make_ts(2025, 12, 18),
                    time_end_utc=make_ts(2025, 12, 28),
                    known_time_step_period="PT8.5M",
                ),
                tasks=[
                    ephem_pb2.EphemTask(
                        task_id=10,
                        sat=common_pb2.Satellite(
                            norad_id=65271,
                            name="X-37B Orbital Test Vehicle 8 (OTV 8)",
                            tle_ln1="1 65271U 25183A   25282.36302114 0.00010000  00000-0  55866-4 0    07",
                            tle_ln2="2 65271  48.7951   8.5514 0002000  85.4867 277.3551 15.78566782    05",
                        ),
                    ),
                ],
            )

            stream = client.Ephem(ephem_req)

            for resp in stream:
                logging.info(
                    "api.v1.Propagator.Ephem stream chunk received:\n"
                    f"ReqId: {resp.req_id}\n"
                    f"TaskId: {resp.result.task_id}\n"
                    f"StreamId: {resp.stream_id}\n"
                    f"StreamChunkId: {resp.stream_chunk_id}\n"
                    f"EphemerisCount: {resp.result.ephem_points_count}"
                )

                for ephem_data in resp.result.ephem_data:
                    logging.info(ephem_data)

            logging.info(
                f"api.v1.Propagator.Ephem done, time took: {time.time() - time_start:.2f}s"
            )

    except grpc.RpcError as e:
        logging.error(f"Failed to request api.v1.Propagator.Ephem: {e}")


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    main()
