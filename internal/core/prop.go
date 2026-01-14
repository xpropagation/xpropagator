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

import "C"
import (
	"context"
	"errors"
	"fmt"
	"time"

	apiv1 "github.com/xpropagation/xpropagator/api/v1/gen"
	"github.com/xpropagation/xpropagator/internal/core_helpers"
	"github.com/xpropagation/xpropagator/internal/dllcore"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (propSrv *PropagationService) Prop(ctx context.Context, req *apiv1.PropRequest) (*apiv1.PropResponse, error) {
	// TODO do we need to lock always, so we get rid of parallel requests ? or only for stateless mode?
	globMu.Lock()
	defer globMu.Unlock()

	if err := validatePropRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	//if propSrv.cfg.StatelessMode {
	//	globMu.Lock()
	//	defer globMu.Unlock()
	//}

	startTime := time.Now()

	satKey, release, err := propSrv.gc.Acquire(ctx, req.GetTask().GetSat().GetTleLn1(), req.GetTask().GetSat().GetTleLn2())
	if err != nil {
		return nil, err
	}
	defer func() {
		if release != nil {
			release()
		}
	}()

	res := &apiv1.EphemerisData{}

	if req.GetTask().GetTimeUtc() != nil {
		req.GetTask().Time = UtcToDS50(req.GetTask().GetTimeUtc().AsTime())
		req.TimeType = apiv1.TimeType_TimeDs50
	}

	err = core_helpers.WithDllCall(ctx, func() int {
		flatArr, rc := dllcore.Sgp4PropAll(satKey, dllcore.TimeType(req.GetTimeType()), req.GetTask().GetTime())
		if rc != 0 || len(flatArr) < 8 {
			return rc
		}
		res = &apiv1.EphemerisData{
			Ds50Time: float64(flatArr[0]),
			MseTime:  float64(flatArr[1]),
			X:        float64(flatArr[2]),
			Y:        float64(flatArr[3]),
			Z:        float64(flatArr[4]),
			Vx:       float64(flatArr[5]),
			Vy:       float64(flatArr[6]),
			Vz:       float64(flatArr[7]),
		}
		return rc
	})
	if err != nil {
		return nil, err
	}

	resp := &apiv1.PropResponse{
		ReqId:  req.GetReqId(),
		Result: res,
	}

	propSrv.logger.Info("analytical propagation done", zap.Duration("time took", time.Since(startTime)))

	//if propSrv.cfg.StatelessMode {
	//	return nil, gc.GlobalGC.RemoveAll(ctx)
	//}

	return resp, nil
}

func validatePropRequest(req *apiv1.PropRequest) error {
	if req.GetTask() == nil {
		return errors.New("task is required")
	}
	if err := validateSatellite(req.GetTask().GetSat()); err != nil {
		return err
	}

	if req.GetTimeType() != apiv1.TimeType_TimeMse && req.GetTimeType() != apiv1.TimeType_TimeDs50 {
		return fmt.Errorf("invalid time type: %v (valid types: MSE, DS50)", req.GetTimeType())
	}
	if req.GetTask().GetTimeUtc() != nil && req.GetTask().GetTime() > 0 {
		return fmt.Errorf("time cannot be given in DS50 or MSE %v, a UTC time already specified", req.GetTask().GetTime())
	}

	if req.GetTask().GetTimeUtc() == nil && req.GetTimeType() != apiv1.TimeType_TimeMse && req.GetTimeType() != apiv1.TimeType_TimeDs50 {
		return fmt.Errorf("time must be specified DS50/MSE or UTC")
	}
	return nil
}
