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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xpropagation/xpropagator/internal/values"
	"go.uber.org/zap"
)

func TestGetEnv_WithValue(t *testing.T) {
	key := "TEST_ENV_KEY_EXISTS"
	expected := "test_value"
	os.Setenv(key, expected)
	defer os.Unsetenv(key)

	result := GetEnv(key, "fallback")
	if result != expected {
		t.Errorf("GetEnv() = %v, want %v", result, expected)
	}
}

func TestGetEnv_WithFallback(t *testing.T) {
	key := "TEST_ENV_KEY_NOT_EXISTS_12345"
	os.Unsetenv(key) // Ensure it doesn't exist

	expected := "fallback_value"
	result := GetEnv(key, expected)
	if result != expected {
		t.Errorf("GetEnv() = %v, want %v", result, expected)
	}
}

func TestGetEnv_EmptyValue(t *testing.T) {
	key := "TEST_ENV_KEY_EMPTY"
	os.Setenv(key, "")
	defer os.Unsetenv(key)

	// Empty string is still a valid value, should not use fallback
	result := GetEnv(key, "fallback")
	if result != "" {
		t.Errorf("GetEnv() = %v, want empty string", result)
	}
}

func TestBuildConfig_Defaults(t *testing.T) {
	// Clear all relevant env vars
	clearEnvVars(t)

	// Use non-existent config file path
	os.Setenv(values.ConfigEnvKey, "/nonexistent/path/config.yaml")
	defer os.Unsetenv(values.ConfigEnvKey)

	logger := zap.NewNop()
	cfg, err := buildConfig(logger)
	if err != nil {
		t.Fatalf("buildConfig() error = %v", err)
	}

	// Check defaults
	if cfg.Reflection != false {
		t.Errorf("Reflection = %v, want false", cfg.Reflection)
	}
	if cfg.StreamChunkSize != values.DefaultStreamChunkSize {
		t.Errorf("StreamChunkSize = %v, want %v", cfg.StreamChunkSize, values.DefaultStreamChunkSize)
	}
	expectedGraceful, _ := time.ParseDuration(values.DefaultGracefulStopTimeoutSec)
	if cfg.GracefulStopTimeoutSec != expectedGraceful {
		t.Errorf("GracefulStopTimeoutSec = %v, want %v", cfg.GracefulStopTimeoutSec, expectedGraceful)
	}
	if cfg.GC.MaxLoadedSatsGc != values.DefaultMaxLoadedSatsGc {
		t.Errorf("MaxLoadedSatsGc = %v, want %v", cfg.GC.MaxLoadedSatsGc, values.DefaultMaxLoadedSatsGc)
	}
	expectedTTL, _ := time.ParseDuration(values.DefaultIdleTTLGcMin)
	if cfg.GC.IdleTTLGcMin != expectedTTL {
		t.Errorf("IdleTTLGcMin = %v, want %v", cfg.GC.IdleTTLGcMin, expectedTTL)
	}
	expectedSweep, _ := time.ParseDuration(values.DefaultSweepIntervalGcMin)
	if cfg.GC.SweepIntervalGcMin != expectedSweep {
		t.Errorf("SweepIntervalGcMin = %v, want %v", cfg.GC.SweepIntervalGcMin, expectedSweep)
	}
	if cfg.TLS.Enabled != false {
		t.Errorf("TLS.Enabled = %v, want false", cfg.TLS.Enabled)
	}
	if cfg.TLS.CertFile != values.DefaultTLSCertFilePath {
		t.Errorf("TLS.CertFile = %v, want %v", cfg.TLS.CertFile, values.DefaultTLSCertFilePath)
	}
	if cfg.TLS.KeyFile != values.DefaultTLSKeyFilePath {
		t.Errorf("TLS.KeyFile = %v, want %v", cfg.TLS.KeyFile, values.DefaultTLSKeyFilePath)
	}
	if cfg.TLS.CAFile != values.DefaultTLSCaFilePath {
		t.Errorf("TLS.CAFile = %v, want %v", cfg.TLS.CAFile, values.DefaultTLSCaFilePath)
	}
}

func TestBuildConfig_FromYAMLFile(t *testing.T) {
	clearEnvVars(t)

	// Create temp config file
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test_config.yaml")
	cfgContent := `
reflection: true
stream_chunk_size: 50
graceful_stop_timeout_sec: 30s
gc:
  max_loaded_sats_gc: 1000
  idle_ttl_gc_min: 15m
  sweep_interval_gc_min: 3m
tls:
  enabled: true
  cert_file: "/custom/cert.pem"
  key_file: "/custom/key.pem"
  ca_file: "/custom/ca.pem"
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	os.Setenv(values.ConfigEnvKey, cfgPath)
	defer os.Unsetenv(values.ConfigEnvKey)

	logger := zap.NewNop()
	cfg, err := buildConfig(logger)
	if err != nil {
		t.Fatalf("buildConfig() error = %v", err)
	}

	if cfg.Reflection != true {
		t.Errorf("Reflection = %v, want true", cfg.Reflection)
	}
	if cfg.StreamChunkSize != 50 {
		t.Errorf("StreamChunkSize = %v, want 50", cfg.StreamChunkSize)
	}
	if cfg.GracefulStopTimeoutSec != 30*time.Second {
		t.Errorf("GracefulStopTimeoutSec = %v, want 30s", cfg.GracefulStopTimeoutSec)
	}
	if cfg.GC.MaxLoadedSatsGc != 1000 {
		t.Errorf("MaxLoadedSatsGc = %v, want 1000", cfg.GC.MaxLoadedSatsGc)
	}
	if cfg.GC.IdleTTLGcMin != 15*time.Minute {
		t.Errorf("IdleTTLGcMin = %v, want 15m", cfg.GC.IdleTTLGcMin)
	}
	if cfg.GC.SweepIntervalGcMin != 3*time.Minute {
		t.Errorf("SweepIntervalGcMin = %v, want 3m", cfg.GC.SweepIntervalGcMin)
	}
	if cfg.TLS.Enabled != true {
		t.Errorf("TLS.Enabled = %v, want true", cfg.TLS.Enabled)
	}
	if cfg.TLS.CertFile != "/custom/cert.pem" {
		t.Errorf("TLS.CertFile = %v, want /custom/cert.pem", cfg.TLS.CertFile)
	}
	if cfg.TLS.KeyFile != "/custom/key.pem" {
		t.Errorf("TLS.KeyFile = %v, want /custom/key.pem", cfg.TLS.KeyFile)
	}
	if cfg.TLS.CAFile != "/custom/ca.pem" {
		t.Errorf("TLS.CAFile = %v, want /custom/ca.pem", cfg.TLS.CAFile)
	}
}

func TestBuildConfig_EnvOverridesYAML(t *testing.T) {
	clearEnvVars(t)

	// Create temp config file with some values
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test_config.yaml")
	cfgContent := `
reflection: false
stream_chunk_size: 50
tls:
  enabled: true
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	os.Setenv(values.ConfigEnvKey, cfgPath)
	defer os.Unsetenv(values.ConfigEnvKey)

	// Set env vars to override YAML values
	os.Setenv(values.ReflectionEnvKey, "true")
	defer os.Unsetenv(values.ReflectionEnvKey)
	os.Setenv(values.StreamChunkSizeEnvKey, "200")
	defer os.Unsetenv(values.StreamChunkSizeEnvKey)
	os.Setenv(values.EnableTLSEnvKey, "false")
	defer os.Unsetenv(values.EnableTLSEnvKey)

	logger := zap.NewNop()
	cfg, err := buildConfig(logger)
	if err != nil {
		t.Fatalf("buildConfig() error = %v", err)
	}

	// Env should override YAML
	if cfg.Reflection != true {
		t.Errorf("Reflection = %v, want true (env override)", cfg.Reflection)
	}
	if cfg.StreamChunkSize != 200 {
		t.Errorf("StreamChunkSize = %v, want 200 (env override)", cfg.StreamChunkSize)
	}
	if cfg.TLS.Enabled != false {
		t.Errorf("TLS.Enabled = %v, want false (env override)", cfg.TLS.Enabled)
	}
}

func TestBuildConfig_AllEnvOverrides(t *testing.T) {
	clearEnvVars(t)

	// Use non-existent config file
	os.Setenv(values.ConfigEnvKey, "/nonexistent/path/config.yaml")
	defer os.Unsetenv(values.ConfigEnvKey)

	// Set all env vars
	os.Setenv(values.ReflectionEnvKey, "true")
	defer os.Unsetenv(values.ReflectionEnvKey)
	os.Setenv(values.StreamChunkSizeEnvKey, "75")
	defer os.Unsetenv(values.StreamChunkSizeEnvKey)
	os.Setenv(values.GracefulStopTimeoutSecEnvKey, "45s")
	defer os.Unsetenv(values.GracefulStopTimeoutSecEnvKey)
	os.Setenv(values.MaxLoadedSatsGcEnvKey, "2000")
	defer os.Unsetenv(values.MaxLoadedSatsGcEnvKey)
	os.Setenv(values.IdleTTLGcMinEnvKey, "25m")
	defer os.Unsetenv(values.IdleTTLGcMinEnvKey)
	os.Setenv(values.SweepIntervalGcMinEnvKey, "7m")
	defer os.Unsetenv(values.SweepIntervalGcMinEnvKey)
	os.Setenv(values.EnableTLSEnvKey, "true")
	defer os.Unsetenv(values.EnableTLSEnvKey)
	os.Setenv(values.TLSCertFilePathEnvKey, "/env/cert.crt")
	defer os.Unsetenv(values.TLSCertFilePathEnvKey)
	os.Setenv(values.TLSKeyFilePathEnvKey, "/env/key.key")
	defer os.Unsetenv(values.TLSKeyFilePathEnvKey)
	os.Setenv(values.TLSCaFilePathEnvKey, "/env/ca.crt")
	defer os.Unsetenv(values.TLSCaFilePathEnvKey)

	logger := zap.NewNop()
	cfg, err := buildConfig(logger)
	if err != nil {
		t.Fatalf("buildConfig() error = %v", err)
	}

	if cfg.Reflection != true {
		t.Errorf("Reflection = %v, want true", cfg.Reflection)
	}
	if cfg.StreamChunkSize != 75 {
		t.Errorf("StreamChunkSize = %v, want 75", cfg.StreamChunkSize)
	}
	if cfg.GracefulStopTimeoutSec != 45*time.Second {
		t.Errorf("GracefulStopTimeoutSec = %v, want 45s", cfg.GracefulStopTimeoutSec)
	}
	if cfg.GC.MaxLoadedSatsGc != 2000 {
		t.Errorf("MaxLoadedSatsGc = %v, want 2000", cfg.GC.MaxLoadedSatsGc)
	}
	if cfg.GC.IdleTTLGcMin != 25*time.Minute {
		t.Errorf("IdleTTLGcMin = %v, want 25m", cfg.GC.IdleTTLGcMin)
	}
	if cfg.GC.SweepIntervalGcMin != 7*time.Minute {
		t.Errorf("SweepIntervalGcMin = %v, want 7m", cfg.GC.SweepIntervalGcMin)
	}
	if cfg.TLS.Enabled != true {
		t.Errorf("TLS.Enabled = %v, want true", cfg.TLS.Enabled)
	}
	if cfg.TLS.CertFile != "/env/cert.crt" {
		t.Errorf("TLS.CertFile = %v, want /env/cert.crt", cfg.TLS.CertFile)
	}
	if cfg.TLS.KeyFile != "/env/key.key" {
		t.Errorf("TLS.KeyFile = %v, want /env/key.key", cfg.TLS.KeyFile)
	}
	if cfg.TLS.CAFile != "/env/ca.crt" {
		t.Errorf("TLS.CAFile = %v, want /env/ca.crt", cfg.TLS.CAFile)
	}
}

func TestBuildConfig_InvalidEnvValues(t *testing.T) {
	clearEnvVars(t)

	// Use non-existent config file
	os.Setenv(values.ConfigEnvKey, "/nonexistent/path/config.yaml")
	defer os.Unsetenv(values.ConfigEnvKey)

	// Set invalid env values
	os.Setenv(values.ReflectionEnvKey, "not_a_bool")
	defer os.Unsetenv(values.ReflectionEnvKey)
	os.Setenv(values.StreamChunkSizeEnvKey, "not_an_int")
	defer os.Unsetenv(values.StreamChunkSizeEnvKey)
	os.Setenv(values.GracefulStopTimeoutSecEnvKey, "not_a_duration")
	defer os.Unsetenv(values.GracefulStopTimeoutSecEnvKey)
	os.Setenv(values.MaxLoadedSatsGcEnvKey, "not_an_int")
	defer os.Unsetenv(values.MaxLoadedSatsGcEnvKey)
	os.Setenv(values.IdleTTLGcMinEnvKey, "not_a_duration")
	defer os.Unsetenv(values.IdleTTLGcMinEnvKey)
	os.Setenv(values.SweepIntervalGcMinEnvKey, "not_a_duration")
	defer os.Unsetenv(values.SweepIntervalGcMinEnvKey)
	os.Setenv(values.EnableTLSEnvKey, "not_a_bool")
	defer os.Unsetenv(values.EnableTLSEnvKey)

	logger := zap.NewNop()
	cfg, err := buildConfig(logger)
	if err != nil {
		t.Fatalf("buildConfig() should not error on invalid env values, got: %v", err)
	}

	// Should fall back to defaults when env values are invalid
	if cfg.Reflection != false {
		t.Errorf("Reflection = %v, want false (default)", cfg.Reflection)
	}
	if cfg.StreamChunkSize != values.DefaultStreamChunkSize {
		t.Errorf("StreamChunkSize = %v, want %v (default)", cfg.StreamChunkSize, values.DefaultStreamChunkSize)
	}
	expectedGraceful, _ := time.ParseDuration(values.DefaultGracefulStopTimeoutSec)
	if cfg.GracefulStopTimeoutSec != expectedGraceful {
		t.Errorf("GracefulStopTimeoutSec = %v, want %v (default)", cfg.GracefulStopTimeoutSec, expectedGraceful)
	}
	if cfg.GC.MaxLoadedSatsGc != values.DefaultMaxLoadedSatsGc {
		t.Errorf("MaxLoadedSatsGc = %v, want %v (default)", cfg.GC.MaxLoadedSatsGc, values.DefaultMaxLoadedSatsGc)
	}
	if cfg.TLS.Enabled != false {
		t.Errorf("TLS.Enabled = %v, want false (default)", cfg.TLS.Enabled)
	}
}

func TestBuildConfig_ZeroAndNegativeValues(t *testing.T) {
	clearEnvVars(t)

	// Use non-existent config file
	os.Setenv(values.ConfigEnvKey, "/nonexistent/path/config.yaml")
	defer os.Unsetenv(values.ConfigEnvKey)

	// Set zero/negative values
	os.Setenv(values.StreamChunkSizeEnvKey, "0")
	defer os.Unsetenv(values.StreamChunkSizeEnvKey)
	os.Setenv(values.GracefulStopTimeoutSecEnvKey, "-5s")
	defer os.Unsetenv(values.GracefulStopTimeoutSecEnvKey)
	os.Setenv(values.MaxLoadedSatsGcEnvKey, "-100")
	defer os.Unsetenv(values.MaxLoadedSatsGcEnvKey)
	os.Setenv(values.IdleTTLGcMinEnvKey, "-10m")
	defer os.Unsetenv(values.IdleTTLGcMinEnvKey)
	os.Setenv(values.SweepIntervalGcMinEnvKey, "0s")
	defer os.Unsetenv(values.SweepIntervalGcMinEnvKey)

	logger := zap.NewNop()
	cfg, err := buildConfig(logger)
	if err != nil {
		t.Fatalf("buildConfig() error = %v", err)
	}

	// Should use defaults for zero/negative values
	if cfg.StreamChunkSize != values.DefaultStreamChunkSize {
		t.Errorf("StreamChunkSize = %v, want %v (default for zero)", cfg.StreamChunkSize, values.DefaultStreamChunkSize)
	}
	expectedGraceful, _ := time.ParseDuration(values.DefaultGracefulStopTimeoutSec)
	if cfg.GracefulStopTimeoutSec != expectedGraceful {
		t.Errorf("GracefulStopTimeoutSec = %v, want %v (default for negative)", cfg.GracefulStopTimeoutSec, expectedGraceful)
	}
	if cfg.GC.MaxLoadedSatsGc != values.DefaultMaxLoadedSatsGc {
		t.Errorf("MaxLoadedSatsGc = %v, want %v (default for negative)", cfg.GC.MaxLoadedSatsGc, values.DefaultMaxLoadedSatsGc)
	}
}

func TestBuildConfig_EmptyTLSPathsIgnored(t *testing.T) {
	clearEnvVars(t)

	// Use non-existent config file
	os.Setenv(values.ConfigEnvKey, "/nonexistent/path/config.yaml")
	defer os.Unsetenv(values.ConfigEnvKey)

	// Set empty TLS paths - should be ignored
	os.Setenv(values.TLSCertFilePathEnvKey, "")
	defer os.Unsetenv(values.TLSCertFilePathEnvKey)
	os.Setenv(values.TLSKeyFilePathEnvKey, "")
	defer os.Unsetenv(values.TLSKeyFilePathEnvKey)
	os.Setenv(values.TLSCaFilePathEnvKey, "")
	defer os.Unsetenv(values.TLSCaFilePathEnvKey)

	logger := zap.NewNop()
	cfg, err := buildConfig(logger)
	if err != nil {
		t.Fatalf("buildConfig() error = %v", err)
	}

	// Empty paths should be ignored, defaults used
	if cfg.TLS.CertFile != values.DefaultTLSCertFilePath {
		t.Errorf("TLS.CertFile = %v, want %v (default, empty ignored)", cfg.TLS.CertFile, values.DefaultTLSCertFilePath)
	}
	if cfg.TLS.KeyFile != values.DefaultTLSKeyFilePath {
		t.Errorf("TLS.KeyFile = %v, want %v (default, empty ignored)", cfg.TLS.KeyFile, values.DefaultTLSKeyFilePath)
	}
	if cfg.TLS.CAFile != values.DefaultTLSCaFilePath {
		t.Errorf("TLS.CAFile = %v, want %v (default, empty ignored)", cfg.TLS.CAFile, values.DefaultTLSCaFilePath)
	}
}

func TestBuildConfig_InvalidYAMLFile(t *testing.T) {
	clearEnvVars(t)

	// Create temp config file with invalid YAML
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "invalid_config.yaml")
	invalidContent := `
reflection: [invalid yaml
  this is not valid:
`
	if err := os.WriteFile(cfgPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	os.Setenv(values.ConfigEnvKey, cfgPath)
	defer os.Unsetenv(values.ConfigEnvKey)

	logger := zap.NewNop()
	_, err := buildConfig(logger)
	if err == nil {
		t.Error("buildConfig() should error on invalid YAML, got nil")
	}
}

func TestBuildConfig_PartialYAMLFile(t *testing.T) {
	clearEnvVars(t)

	// Create temp config file with only some values
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "partial_config.yaml")
	cfgContent := `
reflection: true
stream_chunk_size: 25
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	os.Setenv(values.ConfigEnvKey, cfgPath)
	defer os.Unsetenv(values.ConfigEnvKey)

	logger := zap.NewNop()
	cfg, err := buildConfig(logger)
	if err != nil {
		t.Fatalf("buildConfig() error = %v", err)
	}

	// Values from YAML
	if cfg.Reflection != true {
		t.Errorf("Reflection = %v, want true", cfg.Reflection)
	}
	if cfg.StreamChunkSize != 25 {
		t.Errorf("StreamChunkSize = %v, want 25", cfg.StreamChunkSize)
	}

	// Values not in YAML should use defaults
	if cfg.GC.MaxLoadedSatsGc != values.DefaultMaxLoadedSatsGc {
		t.Errorf("MaxLoadedSatsGc = %v, want %v (default)", cfg.GC.MaxLoadedSatsGc, values.DefaultMaxLoadedSatsGc)
	}
	if cfg.TLS.Enabled != false {
		t.Errorf("TLS.Enabled = %v, want false (default)", cfg.TLS.Enabled)
	}
}

func TestConfig_String(t *testing.T) {
	cfg := &Config{
		Reflection:             true,
		StreamChunkSize:        100,
		GracefulStopTimeoutSec: 10 * time.Second,
		GC: &GCConfig{
			MaxLoadedSatsGc:    500,
			IdleTTLGcMin:       10 * time.Minute,
			SweepIntervalGcMin: 5 * time.Minute,
		},
		TLS: &TLSConfig{
			Enabled:  true,
			CertFile: "cert.crt",
			KeyFile:  "key.key",
			CAFile:   "ca.crt",
		},
	}

	str := cfg.String()

	// Check that important values are in the string
	if str == "" {
		t.Error("Config.String() returned empty string")
	}

	// Check for presence of key information
	expectedSubstrings := []string{
		"grpc reflection mode: true",
		"stream chunk: 100",
		"graceful stop timeout: 10s",
		"max loaded sats gc: 500",
	}

	for _, substr := range expectedSubstrings {
		if !containsSubstring(str, substr) {
			t.Errorf("Config.String() = %q, expected to contain %q", str, substr)
		}
	}
}

func TestTLSConfig_EnableDisable(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		yamlEnabled bool
		wantEnabled bool
	}{
		{
			name:        "env false overrides yaml true",
			envValue:    "false",
			yamlEnabled: true,
			wantEnabled: false,
		},
		{
			name:        "env true overrides yaml false",
			envValue:    "true",
			yamlEnabled: false,
			wantEnabled: true,
		},
		{
			name:        "env 0 means false",
			envValue:    "0",
			yamlEnabled: true,
			wantEnabled: false,
		},
		{
			name:        "env 1 means true",
			envValue:    "1",
			yamlEnabled: false,
			wantEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnvVars(t)

			// Create temp config file
			tmpDir := t.TempDir()
			cfgPath := filepath.Join(tmpDir, "test_config.yaml")
			cfgContent := "tls:\n  enabled: " + boolToString(tt.yamlEnabled) + "\n"
			if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
				t.Fatalf("Failed to write test config file: %v", err)
			}

			os.Setenv(values.ConfigEnvKey, cfgPath)
			defer os.Unsetenv(values.ConfigEnvKey)
			os.Setenv(values.EnableTLSEnvKey, tt.envValue)
			defer os.Unsetenv(values.EnableTLSEnvKey)

			logger := zap.NewNop()
			cfg, err := buildConfig(logger)
			if err != nil {
				t.Fatalf("buildConfig() error = %v", err)
			}

			if cfg.TLS.Enabled != tt.wantEnabled {
				t.Errorf("TLS.Enabled = %v, want %v", cfg.TLS.Enabled, tt.wantEnabled)
			}
		})
	}
}

func TestGCConfig_DurationParsing(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		check    func(*Config) bool
	}{
		{
			name:     "idle TTL with minutes",
			envKey:   values.IdleTTLGcMinEnvKey,
			envValue: "30m",
			check:    func(c *Config) bool { return c.GC.IdleTTLGcMin == 30*time.Minute },
		},
		{
			name:     "idle TTL with hours",
			envKey:   values.IdleTTLGcMinEnvKey,
			envValue: "2h",
			check:    func(c *Config) bool { return c.GC.IdleTTLGcMin == 2*time.Hour },
		},
		{
			name:     "sweep interval with seconds",
			envKey:   values.SweepIntervalGcMinEnvKey,
			envValue: "90s",
			check:    func(c *Config) bool { return c.GC.SweepIntervalGcMin == 90*time.Second },
		},
		{
			name:     "graceful stop with milliseconds",
			envKey:   values.GracefulStopTimeoutSecEnvKey,
			envValue: "5000ms",
			check:    func(c *Config) bool { return c.GracefulStopTimeoutSec == 5*time.Second },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnvVars(t)

			os.Setenv(values.ConfigEnvKey, "/nonexistent/path/config.yaml")
			defer os.Unsetenv(values.ConfigEnvKey)
			os.Setenv(tt.envKey, tt.envValue)
			defer os.Unsetenv(tt.envKey)

			logger := zap.NewNop()
			cfg, err := buildConfig(logger)
			if err != nil {
				t.Fatalf("buildConfig() error = %v", err)
			}

			if !tt.check(cfg) {
				t.Errorf("Duration parsing failed for %s=%s", tt.envKey, tt.envValue)
			}
		})
	}
}

// Helper functions

func clearEnvVars(t *testing.T) {
	t.Helper()
	envVars := []string{
		values.ConfigEnvKey,
		values.ReflectionEnvKey,
		values.StreamChunkSizeEnvKey,
		values.GracefulStopTimeoutSecEnvKey,
		values.MaxLoadedSatsGcEnvKey,
		values.IdleTTLGcMinEnvKey,
		values.SweepIntervalGcMinEnvKey,
		values.EnableTLSEnvKey,
		values.TLSCertFilePathEnvKey,
		values.TLSKeyFilePathEnvKey,
		values.TLSCaFilePathEnvKey,
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
