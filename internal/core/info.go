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
	apiv1 "github.com/xpropagation/xpropagator/api/v1/gen"
	"github.com/xpropagation/xpropagator/internal/dllcore"
	"github.com/xpropagation/xpropagator/internal/values"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

func (propSrv *PropagationService) Info(_ context.Context, _ *emptypb.Empty) (*apiv1.InfoResponse, error) {
	return &apiv1.InfoResponse{
		Name:            values.ServiceName,
		Version:         values.Version,
		Commit:          values.CommitHash,
		BuildDate:       values.BuildDate,
		AstroStdLibInfo: dllcore.DllMainGetInfo(),
		Sgp4LibInfo:     dllcore.Sgp4GetInfo(),
		Timestamp:       timestamppb.New(time.Now().UTC()),
	}, nil
}
