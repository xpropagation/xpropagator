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

package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/xpropagation/xpropagator/internal/values"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var Module = fx.Module("config",
	fx.Provide(NewAppConfig),
)

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	CAFile   string `yaml:"ca_file"`
}

type GCConfig struct {
	MaxLoadedSatsGc    int           `yaml:"max_loaded_sats_gc"`
	IdleTTLGcMin       time.Duration `yaml:"idle_ttl_gc_min"`
	SweepIntervalGcMin time.Duration `yaml:"sweep_interval_gc_min"`
}

type Config struct {
	Reflection             bool          `yaml:"reflection"`
	StreamChunkSize        int           `yaml:"stream_chunk_size"`
	GracefulStopTimeoutSec time.Duration `yaml:"graceful_stop_timeout_sec"`
	GC                     *GCConfig     `yaml:"gc"`
	TLS                    *TLSConfig    `yaml:"tls"`
}

func (c *Config) String() string {
	return fmt.Sprintf("grpc reflection mode: %v, "+
		"stream chunk: %d, graceful stop timeout: %v, max loaded sats gc: %d, idle TTL gc: %v, sweep interval gc: %v",
		c.Reflection, c.StreamChunkSize, c.GracefulStopTimeoutSec,
		c.GC.MaxLoadedSatsGc, c.GC.IdleTTLGcMin, c.GC.SweepIntervalGcMin)
}

func GetEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func NewAppConfig(logger *zap.Logger, lc fx.Lifecycle) *Config {
	cfg, err := buildConfig(logger)
	if err != nil {
		panic(fmt.Sprintf("failed to build config: %v", err))
	}

	// Lifecycle hooks
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("config loaded", zap.Stringer("config", cfg))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("config provider stopped")
			return nil
		},
	})

	return cfg
}

func buildConfig(l *zap.Logger) (*Config, error) {
	cfg := &Config{
		Reflection:      false,
		StreamChunkSize: values.DefaultStreamChunkSize,
		GC: &GCConfig{
			MaxLoadedSatsGc: values.DefaultMaxLoadedSatsGc,
		},
		TLS: &TLSConfig{
			Enabled:  false,
			CertFile: values.DefaultTLSCertFilePath,
			KeyFile:  values.DefaultTLSKeyFilePath,
			CAFile:   values.DefaultTLSCaFilePath,
		},
	}

	cfg.GracefulStopTimeoutSec, _ = time.ParseDuration(values.DefaultGracefulStopTimeoutSec)
	cfg.GC.IdleTTLGcMin, _ = time.ParseDuration(values.DefaultIdleTTLGcMin)
	cfg.GC.SweepIntervalGcMin, _ = time.ParseDuration(values.DefaultSweepIntervalGcMin)

	cfgPath := GetEnv(values.ConfigEnvKey, values.DefaultServiceConfig)
	l.Info("using config file", zap.String("path", cfgPath))

	if f, err := os.Open(cfgPath); err == nil {
		defer f.Close()
		if err = yaml.NewDecoder(f).Decode(cfg); err != nil {
			return nil, fmt.Errorf("parse config file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("open config file: %w", err)
	} else {
		l.Warn("config file not found, using defaults + env overrides")
	}

	// ENV overrides + defaults
	if v, ok := os.LookupEnv(values.ReflectionEnvKey); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Reflection = b
		} else {
			l.Warn("invalid bool in env, ignoring override", zap.String("key", values.ReflectionEnvKey), zap.String("raw value from env", v))
		}
	}

	if v, ok := os.LookupEnv(values.StreamChunkSizeEnvKey); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.StreamChunkSize = n
		} else {
			l.Warn("invalid int in env, ignoring override", zap.String("key", values.StreamChunkSizeEnvKey), zap.String("raw value from env", v))
		}
	}
	if cfg.StreamChunkSize <= 0 {
		cfg.StreamChunkSize = values.DefaultStreamChunkSize
	}

	if v, ok := os.LookupEnv(values.GracefulStopTimeoutSecEnvKey); ok {
		if dur, err := time.ParseDuration(v); err == nil {
			cfg.GracefulStopTimeoutSec = dur
		} else {
			l.Warn("invalid str(duration) in env, ignoring override", zap.String("key", values.GracefulStopTimeoutSecEnvKey), zap.String("raw value from env", v))
		}
	}
	if cfg.GracefulStopTimeoutSec <= 0 {
		cfg.GracefulStopTimeoutSec, _ = time.ParseDuration(values.DefaultGracefulStopTimeoutSec)
	}

	if v, ok := os.LookupEnv(values.MaxLoadedSatsGcEnvKey); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.GC.MaxLoadedSatsGc = n
		} else {
			l.Warn("invalid int in env, ignoring override", zap.String("key", values.MaxLoadedSatsGcEnvKey), zap.String("raw value from env", v))
		}
	}
	if cfg.GC.MaxLoadedSatsGc <= 0 {
		cfg.GC.MaxLoadedSatsGc = values.DefaultMaxLoadedSatsGc
	}

	if v, ok := os.LookupEnv(values.IdleTTLGcMinEnvKey); ok {
		if dur, err := time.ParseDuration(v); err == nil {
			cfg.GC.IdleTTLGcMin = dur
		} else {
			l.Warn("invalid str(duration) in env, ignoring override", zap.String("key", values.IdleTTLGcMinEnvKey), zap.String("raw value from env", v))
		}
	}
	if cfg.GC.IdleTTLGcMin <= 0 {
		cfg.GC.IdleTTLGcMin, _ = time.ParseDuration(values.DefaultIdleTTLGcMin)
	}

	if v, ok := os.LookupEnv(values.SweepIntervalGcMinEnvKey); ok {
		if dur, err := time.ParseDuration(v); err == nil {
			cfg.GC.SweepIntervalGcMin = dur
		} else {
			l.Warn("invalid str(duration) in env, ignoring override", zap.String("key", values.SweepIntervalGcMinEnvKey), zap.String("raw value from env", v))
		}
	}
	if cfg.GC.SweepIntervalGcMin <= 0 {
		cfg.GC.IdleTTLGcMin, _ = time.ParseDuration(values.DefaultSweepIntervalGcMin)
	}

	if v, ok := os.LookupEnv(values.EnableTLSEnvKey); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.TLS.Enabled = b
		} else {
			l.Warn("invalid bool in env, ignoring override", zap.String("key", values.EnableTLSEnvKey), zap.String("raw value from env", v))
		}
	}

	if v, ok := os.LookupEnv(values.TLSCertFilePathEnvKey); ok {
		if v != "" {
			cfg.TLS.CertFile = v
		} else {
			l.Warn("invalid str(empty) in env, ignoring override", zap.String("key", values.TLSCertFilePathEnvKey), zap.String("raw value from env", v))
		}
	}

	if v, ok := os.LookupEnv(values.TLSKeyFilePathEnvKey); ok {
		if v != "" {
			cfg.TLS.KeyFile = v
		} else {
			l.Warn("invalid str(empty) in env, ignoring override", zap.String("key", values.TLSKeyFilePathEnvKey), zap.String("raw value from env", v))
		}
	}

	if v, ok := os.LookupEnv(values.TLSCaFilePathEnvKey); ok {
		if v != "" {
			cfg.TLS.CAFile = v
		} else {
			l.Warn("invalid str(empty) in env, ignoring override", zap.String("key", values.TLSCaFilePathEnvKey), zap.String("raw value from env", v))
		}
	}

	return cfg, nil
}
