#!/usr/bin/env bash
#
# MIT License
#
# Copyright (c) 2026 Roman Bielyi
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.
#

set -e

if [ "$#" -ne 2 ]; then
  echo "Usage: $0 <proto_root_dir> <output_dir>"
  exit 1
fi

PROTO_ROOT=$1
OUT_DIR=$2

echo "==> Cleaning old generated files..."
rm -rf api

python -m grpc_tools.protoc \
  -I "$PROTO_ROOT" \
  --python_out="$OUT_DIR" \
  --grpc_python_out="$OUT_DIR" \
  api/v1/main.proto \
  api/v1/info.proto \
  api/v1/common.proto \
  api/v1/core/ephem.proto \
  api/v1/core/file.proto \
  api/v1/core/prop.proto

echo "==> Creating __init__.py files..."
find "$OUT_DIR" -type d -exec touch {}/__init__.py \;

echo "✅ Generation complete!"
echo "Python files generated in: $OUT_DIR"
