/*
 * MIT License
 *
 * Copyright (c) 2026 Roman Bielyi
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package values

var (
	Version    string
	CommitHash string
	BuildDate  string
)

const ServiceName = "XPropagator Server"

const (
	DefaultServiceHost            = "0.0.0.0"
	DefaultServicePort            = "50051"
	DefaultServiceConfig          = "config/cfg_default.yaml"
	DefaultStreamChunkSize        = 100
	DefaultGracefulStopTimeoutSec = "10s"
	DefaultMaxLoadedSatsGc        = 500
	DefaultIdleTTLGcMin           = "10m"
	DefaultSweepIntervalGcMin     = "5m"
	DefaultTLSCertFilePath        = "certs/server.crt"
	DefaultTLSKeyFilePath         = "certs/server.key"
	DefaultTLSCaFilePath          = "certs/ca.crt"
)

const (
	HostEnvKey                   = "SERVICE_HOST"
	PortEnvKey                   = "SERVICE_PORT"
	ConfigEnvKey                 = "SERVICE_CONFIG"
	EnableTLSEnvKey              = "SERVICE_ENABLE_TLS"
	TLSCertFilePathEnvKey        = "SERVICE_TLS_CERT_FILE_PATH"
	TLSKeyFilePathEnvKey         = "SERVICE_TLS_KEY_FILE_PATH"
	TLSCaFilePathEnvKey          = "SERVICE_TLS_CA_FILE_PATH"
	ReflectionEnvKey             = "SERVICE_REFLECTION"
	StreamChunkSizeEnvKey        = "SERVICE_STREAM_CHUNK_SIZE"
	GracefulStopTimeoutSecEnvKey = "SERVICE_GRACEFUL_STOP_TIMEOUT_SEC"
	MaxLoadedSatsGcEnvKey        = "SERVICE_MAX_LOADED_SATS_GC"
	IdleTTLGcMinEnvKey           = "SERVICE_IDLE_TTL_GC_MIN"
	SweepIntervalGcMinEnvKey     = "SERVICE_SWEEP_INTERVAL_GC_MIN"
)
