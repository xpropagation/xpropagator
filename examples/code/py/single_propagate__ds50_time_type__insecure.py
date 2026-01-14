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

from api.v1.core import prop_pb2
from api.v1 import common_pb2
from api.v1 import main_pb2_grpc

def main():
    try:
        with grpc.insecure_channel("localhost:50051") as channel:
            client = main_pb2_grpc.PropagatorStub(channel)

            time_start = time.time()

            prop_req = prop_pb2.PropRequest(
                req_id=1,
                time_type=prop_pb2.TimeType.TimeDs50,
                task=prop_pb2.PropTask(
                    time=27744.5,
                    sat=common_pb2.Satellite(
                        norad_id=65271,
                        name="X-37B Orbital Test Vehicle 8 (OTV 8)",
                        tle_ln1="1 65271U 25183A   25282.36302114 0.00010000  00000-0  55866-4 0    07",
                        tle_ln2="2 65271  48.7951   8.5514 0002000  85.4867 277.3551 15.78566782    05",
                    ),
                ),
            )

            resp = client.Prop(prop_req)

            logging.info(
                "api.v1.Propagator.Prop done, time took: %.2fs\n%s",
                time.time() - time_start,
                resp.result
            )

    except grpc.RpcError as e:
        logging.error(f"Failed to request api.v1.Propagator.Prop: {e}")


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    main()
