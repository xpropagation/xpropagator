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

package core

import (
	"context"
	"fmt"
	"sync"

	apiv1 "github.com/xpropagation/xpropagator/api/v1/gen"
	"github.com/xpropagation/xpropagator/internal/config"
	"github.com/xpropagation/xpropagator/internal/core/gc"
	"github.com/xpropagation/xpropagator/internal/core_helpers"
	"github.com/xpropagation/xpropagator/internal/dllcore"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/proto"
)

var globErrMu sync.Mutex
var globMu sync.Mutex

func validateSatellite(sat *apiv1.Satellite) error {
	if sat == nil {
		return fmt.Errorf("task must include a satellite")
	}
	if sat.GetTleLn1() == "" {
		return fmt.Errorf("satellite is missing TLE line 1")
	}
	if sat.GetTleLn2() == "" {
		return fmt.Errorf("satellite is missing TLE line 2")
	}

	return nil
}

func getDllLastError() string {
	globErrMu.Lock()
	defer globErrMu.Unlock()
	return dllcore.GetLastErrMsg()
}

func LoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		p, _ := peer.FromContext(ctx)

		l := logger

		l.Info("[RPC Unary]", zap.String("client", p.Addr.String()), zap.String("method", info.FullMethod), zap.Any("request", req))

		resp, err := handler(ctx, req)
		if err != nil {
			l.Error("[RPC Unary]", zap.String("client", p.Addr.String()), zap.String("method", info.FullMethod), zap.Error(err))
		} else {
			g := resp.(proto.Message)
			l.Info("[RPC Unary]", zap.String("client", p.Addr.String()), zap.String("method", info.FullMethod), core_helpers.ProtoField("response", g))
		}

		return resp, err
	}
}

func LoggingStreamInterceptor(logger *zap.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		p, _ := peer.FromContext(ss.Context())

		logger.Info("[RPC Stream Start]",
			zap.String("client", p.Addr.String()),
			zap.String("method", info.FullMethod),
			zap.Bool("client_stream", info.IsClientStream),
			zap.Bool("server_stream", info.IsServerStream))

		err := handler(srv, ss)

		if err != nil {
			logger.Error("[RPC Stream End]",
				zap.String("client", p.Addr.String()),
				zap.String("method", info.FullMethod),
				zap.Error(err))
		} else {
			logger.Info("[RPC Stream End]",
				zap.String("client", p.Addr.String()),
				zap.String("method", info.FullMethod))
		}

		return err
	}
}

type PropagationService struct {
	apiv1.UnimplementedPropagatorServer
	cfg    *config.Config
	logger *zap.Logger
	gc     *gc.GC
}
