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

from google.protobuf.empty_pb2 import Empty
from api.v1 import main_pb2_grpc
import grpc
import logging

def main():
    try:
        with grpc.insecure_channel("localhost:50051") as channel:
            client = main_pb2_grpc.PropagatorStub(channel)

            response = client.Info(Empty())

            logging.info(
                "api.v1.Propagator.Info response:\n"
                f"Name: {response.name}\n"
                f"Version: {response.version}\n"
                f"Commit: {response.commit}\n"
                f"BuildDate: {response.build_date}\n"
                f"AstroStdLibInfo: {response.astro_std_lib_info}\n"
                f"Sgp4LibInfo: {response.sgp4_lib_info}\n"
                f"Timestamp: {response.timestamp.ToDatetime()}"
            )

    except grpc.RpcError as e:
        logging.error(f"Failed to request api.v1.Propagator.Info: {e}")

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    main()

