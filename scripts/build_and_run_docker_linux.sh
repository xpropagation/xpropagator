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

set -e

# Usage:
#   ./build_and_run_docker_linux.sh <saal_lib_path> <saal_wrappers_path> [true|false]
#
# Arguments:
#   saal_lib_path      - Path to SAAL library files
#   saal_wrappers_path - Path to SAAL wrapper header files
#   tls_enabled        - Optional: 'true' to enable TLS, 'false' to disable (default)

if [ "$#" -lt 2 ] || [ "$#" -gt 3 ]; then
  echo "Usage: $0 <saal_lib_path> <saal_wrappers_path> [true|false]"
  exit 1
fi

SGP4_LIB_PATH=$1
SAAL_WRAPPERS_PATH=$2
TLS_ENABLED=${3:-false}

echo ">>> Cleaning old state..."
rm -rf ./step ./certs ~/.step

echo ">>> Preparing SAAL libs for image..."

TAG=""
if [[ "$SGP4_LIB_PATH" == *"Linux_ARM"* ]]; then
  TAG="linux_arm64"
elif [[ "$SGP4_LIB_PATH" == *"Linux/GFORTRAN"* ]]; then
  TAG="linux_amd64"
else
  echo "❌ Could not determine architecture from SGP4_LIB_PATH: $SAAL_LIB_PATH"
  echo "Expected path to contain either 'Linux_ARM' or 'Linux/GFORTRAN'"
  exit 1
fi

export CGO_ENABLED=1
export TLS_ENABLED=$TLS_ENABLED

sudo xattr -r -d com.apple.quarantine "${SGP4_LIB_PATH}" 2>/dev/null || true

mkdir -p ../internal/dllcore/_lib ../internal/dllcore/_wrappers

LIB_FILES=(
  libelops.so
  libgfortran.so.5
  libtimefunc.so
  libenvconst.so
  libgomp.so.1
  libtle.so
  libastrofunc.so
  libsgp4prop.so
  libdllmain.so
  libgcc_s.so.1
  SGP4_Open_License.txt
)

if [[ "$TAG" == "linux_amd64" ]]; then
  LIB_FILES+=("ld-linux-x86-64.so.2" "libquadmath.so.0" "libz.so.1")
fi

WRAPPER_FILES=(
  AstroFuncDll.h
  DllMainDll.h
  EnvConstDll.h
  Sgp4PropDll.h
  TimeFuncDll.h
  TleDll.h
)

for f in "${LIB_FILES[@]}"; do
  if [ -f "${SGP4_LIB_PATH}/${f}" ]; then
    cp "${SGP4_LIB_PATH}/${f}" ../internal/dllcore/_lib/
  else
    echo "⚠️  Warning: ${SGP4_LIB_PATH}/${f} not found"
  fi
done

for f in "${WRAPPER_FILES[@]}"; do
  if [ -f "${SAAL_WRAPPERS_PATH}/${f}" ]; then
    cp "${SAAL_WRAPPERS_PATH}/${f}" ../internal/dllcore/_wrappers/
  else
    echo "⚠️  Warning: ${SAAL_WRAPPERS_PATH}/${f} not found"
  fi
done

COMMIT=$(git rev-parse HEAD)
BUILD=$(date -u +%Y-%m-%dT%H:%M:%SZ)

echo ">>> Building image with tag xpropagator-server:${TAG} ..."
echo "    TLS enabled: ${TLS_ENABLED}"
echo "    Commit: ${COMMIT}"
echo "    Build:  ${BUILD}"

docker buildx build \
  --progress=plain \
  --load \
  --build-arg COMMIT="${COMMIT}" \
  --build-arg BUILD="${BUILD}" \
  --build-arg SGP4_LIB_PATH="${SGP4_LIB_PATH}" \
  --no-cache \
  -t xpropagator-server:${TAG} ../. 2>&1

rm -rf ../internal/dllcore/_lib ../internal/dllcore/_wrappers

export TAG

if [ "$TLS_ENABLED" = "true" ]; then
  echo "🔒 SECURE mode - generating TLS certificates"
  
  echo ">>> Generating certificates with local scripts..."
  
  sh ./tls_init_and_run.sh &
  sleep 10
  
  sh ./tls_gen_ca_local.sh || { echo "❌ CA failed"; exit 1; }
  sh ./tls_gen_server_local.sh || echo "⚠️ Server failed"
  sh ./tls_gen_client_local.sh || echo "⚠️ Client failed"
  
  sleep 5
  pkill -f "step-ca" || true
  
  sudo chown -R $(whoami) certs/ ../config/ 2>/dev/null || true
  
  echo ">>> Starting container with image xpropagator-server:${TAG} ..."
  docker run -d \
    --name xpropagator-server \
    -p 50051:50051 \
    -v $(pwd)/certs:/certs:ro \
    -v $(pwd)/../config:/config:ro \
    -e SERVICE_TLS_CERT_FILE_PATH=/certs/server.crt \
    -e SERVICE_TLS_KEY_FILE_PATH=/certs/server.key \
    -e SERVICE_TLS_CA_FILE_PATH=/certs/ca.crt \
    -e SERVICE_ENABLE_TLS=true \
    -e SERVICE_CONFIG=cfg.yaml \
    -e SERVICE_REFLECTION=true \
    --restart unless-stopped \
    xpropagator-server:${TAG}

  echo ">>> Done."
  echo "    - XPropagator Server:   localhost:50051"
  echo "    - certs:          ./certs/{ca,server,client}.*"
  echo "    - XPropagator image:    xpropagator-server:${TAG}"

else
  echo "🔓 INSECURE mode - skipping TLS certificate generation"
  
  echo ">>> Starting container with image xpropagator-server:${TAG} (INSECURE - no TLS)..."
  docker run -d \
    --name xpropagator-server \
    -p 50051:50051 \
    -v $(pwd)/../config:/config:ro \
    -e SERVICE_ENABLE_TLS=false \
    -e SERVICE_REFLECTION=true \
    --restart unless-stopped \
    xpropagator-server:${TAG}

  echo ">>> Done."
  echo "    - XPropagator Server:   localhost:50051 (INSECURE - no TLS)"
  echo "    - XPropagator image:    xpropagator-server:${TAG}"
fi

echo ""
echo "Stop with: docker stop xpropagator-server && docker rm xpropagator-server"
