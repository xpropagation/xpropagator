#!/bin/bash

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

# Unit tests for GC package
# Requires SGP4 libraries to be installed in /usr/local/lib/
# Run build_local_darwin.sh or build_local_linux.sh first to install libraries

echo "🚀 Running unit tests for GC package..."
echo "📍 Libraries expected in: /usr/local/lib/"

# Set library path based on platform
if [[ "$(uname)" == "Darwin" ]]; then
  export DYLD_LIBRARY_PATH="/usr/local/lib:${DYLD_LIBRARY_PATH:-}"
  echo "📍 DYLD_LIBRARY_PATH set for macOS"
else
  export LD_LIBRARY_PATH="/usr/local/lib:${LD_LIBRARY_PATH:-}"
  echo "📍 LD_LIBRARY_PATH set for Linux"
fi

if ! go test ../internal/core/gc -v; then
  echo "❌ Tests FAILED"
  exit 1
else
  echo "✅ All tests PASSED!"
fi
