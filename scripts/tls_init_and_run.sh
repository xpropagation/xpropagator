#!/bin/sh
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

STEPPATH="${STEPPATH:-${HOME}/.step}"
export STEPPATH

mkdir -p "${STEPPATH}"

if [ ! -f "${STEPPATH}/ca-pass.txt" ]; then
  echo 'super-secret-ca-pass' > "${STEPPATH}/ca-pass.txt"
  chmod 600 "${STEPPATH}/ca-pass.txt"
fi

if [ ! -f "${STEPPATH}/prov-pass.txt" ]; then
  echo 'super-secret-prov-pass' > "${STEPPATH}/prov-pass.txt"
  chmod 600 "${STEPPATH}/prov-pass.txt"
fi

if [ ! -f "${STEPPATH}/config/ca.json" ]; then
  step ca init \
    --deployment-type standalone \
    --name "Dev CA" \
    --dns "step-ca,localhost" \
    --address ":9000" \
    --provisioner "dev" \
    --password-file "${STEPPATH}/ca-pass.txt" \
    --provisioner-password-file "${STEPPATH}/prov-pass.txt" \
    --acme=false \
    --remote-management=false
fi

step-ca "${STEPPATH}/config/ca.json" --password-file "${STEPPATH}/ca-pass.txt"
