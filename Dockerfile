# ---------- Stage 1: Build ----------
FROM golang:1.23-bookworm AS builder

WORKDIR /app
ENV CGO_ENABLED=1

# Install build tools and gfortran runtime for linking
RUN apt-get update && apt-get install -y build-essential gfortran

# Copy source code
COPY . .

RUN go mod download

# Build metadata
ARG VERSION=v1.0.0
ARG COMMIT
ARG BUILD

# Install SGP4 libraries and license file for tests and runtime
RUN cp /app/internal/dllcore/_lib/*.so /usr/local/lib/ 2>/dev/null || true && \
    cp /app/internal/dllcore/_lib/SGP4_Open_License.txt /usr/local/lib/ 2>/dev/null || true && \
    cp /app/internal/dllcore/_lib/SGP4_Open_License.txt /app/ 2>/dev/null || true && \
    cp /app/internal/dllcore/_lib/SGP4_Open_License.txt /app/scripts/ 2>/dev/null || true && \
    cp /app/internal/dllcore/_lib/SGP4_Open_License.txt /app/internal/core/ 2>/dev/null || true && \
    cp /app/internal/dllcore/_lib/SGP4_Open_License.txt /app/internal/core/gc/ 2>/dev/null || true && \
    ldconfig

# Run tests
WORKDIR /app/scripts
RUN bash ./run_unit_tests_config.sh
RUN bash ./run_unit_tests_gc.sh
RUN bash ./run_integration_tests_core.sh
WORKDIR /app

# Build Go binary
RUN go build -ldflags="-s -w \
    -X github.com/xpropagation/xpropagator/internal/values.Version=${VERSION} \
    -X github.com/xpropagation/xpropagator/internal/values.CommitHash=${COMMIT} \
    -X github.com/xpropagation/xpropagator/internal/values.BuildDate=${BUILD}" \
    -o bin/xpropagator .

# ---------- Stage 2: Runtime ----------
FROM debian:bookworm-slim

WORKDIR /app

# Install Fortran runtime for execution
RUN apt-get update && apt-get install -y libgfortran5 && rm -rf /var/lib/apt/lists/*

# Copy Go binary and shared libraries from builder
COPY --from=builder /app/bin/xpropagator .
COPY --from=builder /app/internal/dllcore/_lib/SGP4_Open_License.txt .
COPY --from=builder /app/config/cfg_default.yaml ./cfg.yaml
COPY --from=builder /app/internal/dllcore/_lib/ /usr/local/lib/

ENV SERVICE_CONFIG=cfg.yaml

RUN ldconfig

ENTRYPOINT ["./xpropagator"]
