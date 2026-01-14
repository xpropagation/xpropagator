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

//go:generate protoc -I . --go_out=. --go-grpc_out=. api/v1/main.proto api/v1/info.proto api/v1/core/ephem.proto api/v1/core/prop.proto api/v1/common.proto
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"os"
	"time"

	apiv1 "github.com/xpropagation/xpropagator/api/v1/gen"
	"github.com/xpropagation/xpropagator/internal/config"
	"github.com/xpropagation/xpropagator/internal/core"
	"github.com/xpropagation/xpropagator/internal/logger"
	"github.com/xpropagation/xpropagator/internal/values"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type GRPCServerParams struct {
	fx.In
	Logger      *zap.Logger
	Config      *config.Config
	PropService *core.PropagationService
}

type GRPCServerResult struct {
	fx.Out
	Server       *grpc.Server
	HealthServer *health.Server
}

func newGRPCServer(params GRPCServerParams) (GRPCServerResult, error) {
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(core.LoggingInterceptor(params.Logger)),
		grpc.StreamInterceptor(core.LoggingStreamInterceptor(params.Logger)),
	}

	if params.Config.TLS.Enabled {
		params.Logger.Info("mTLS is enabled")
		tlsCfg, err := buildTLSConfig(params.Config)
		if err != nil {
			return GRPCServerResult{}, fmt.Errorf("failed to build TLS config: %w", err)
		}
		creds := credentials.NewTLS(tlsCfg)
		opts = append(opts, grpc.Creds(creds))
	} else {
		opts = append(opts, grpc.Creds(insecure.NewCredentials()))
	}

	server := grpc.NewServer(opts...)

	apiv1.RegisterPropagatorServer(server, params.PropService)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	if params.Config.Reflection {
		params.Logger.Info("gRPC server reflection enabled")
		reflection.Register(server)
	}

	params.Logger.Info("gRPC server created successfully")

	return GRPCServerResult{
		Server:       server,
		HealthServer: healthServer,
	}, nil
}

func newListener(logger *zap.Logger) (net.Listener, error) {
	addr := fmt.Sprintf("%s:%s",
		config.GetEnv(values.HostEnvKey, values.DefaultServiceHost),
		config.GetEnv(values.PortEnvKey, values.DefaultServicePort),
	)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	logger.Info("TCP listener created", zap.String("addr", addr))
	return lis, nil
}

type GRPCLifecycleParams struct {
	fx.In

	Logger       *zap.Logger
	Server       *grpc.Server
	Listener     net.Listener
	Config       *config.Config
	HealthServer *health.Server
	Lifecycle    fx.Lifecycle
}

func registerGRPCLifecycle(params GRPCLifecycleParams) {
	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			params.Logger.Info("starting XPropagator gRPC server")
			go func() {
				if err := params.Server.Serve(params.Listener); err != nil {
					params.Logger.Error("gRPC server error", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			params.Logger.Info("shutting down XPropagator gRPC server")

			params.HealthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)

			done := make(chan struct{})
			go func() {
				params.Server.GracefulStop()
				close(done)
			}()

			select {
			case <-done:
				params.Logger.Info("gRPC server stopped gracefully")
			case <-ctx.Done():
				params.Logger.Warn("context cancelled during shutdown")
				params.Server.Stop()
			case <-time.After(params.Config.GracefulStopTimeoutSec):
				params.Logger.Warn("force stopping gRPC server after timeout")
				params.Server.Stop()
			}

			params.Logger.Info("XPropagator gRPC server has been successfully shutdown")
			return nil
		},
	})
}

func startApp() {
	printBanner()
}

func buildTLSConfig(cfg *config.Config) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate key pair: %v", err)
	}

	caCertPEM, err := os.ReadFile(cfg.TLS.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %v", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func printBanner() {
	fmt.Println("┌──────────────────────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│  " + values.ServiceName + " " + values.Version + "                                                                   │")
	fmt.Println("├──────────────────────────────────────────────────────────────────────────────────────────────┤")
	fmt.Println("│  🛰️  Modern Satellite Orbit Propagation as a Service.                                         │")
	fmt.Println("│  Powerful gRPC API for 🇺🇸 ultra‑precise U.S. Space Force (USSF) SGP4 / SGP4‑XP propagator.   │")
	fmt.Println("│  Catalog‑scale satellite processing, ephemeris generation, and built‑in memory management.   │")
	fmt.Println("│  Written in Go                                                                               │")
	fmt.Println("└──────────────────────────────────────────────────────────────────────────────────────────────┘")

}

func main() {
	printBuild := flag.Bool("print-build", false, "print build information")
	flag.Parse()

	if *printBuild {
		l, _ := logger.NewLogger()
		l.Info("build info",
			zap.String("version", values.Version),
			zap.String("commit", values.CommitHash),
			zap.String("date", values.BuildDate),
		)
		os.Exit(0)
	}

	app := fx.New(
		config.Module,
		logger.Module,
		core.Module,
		fx.Provide(
			newGRPCServer,
			newListener,
		),
		fx.Invoke(startApp),
		fx.Invoke(registerGRPCLifecycle),
	)

	app.Run()
}
