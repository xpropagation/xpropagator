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
	"fmt"
	"time"

	"github.com/sosodev/duration"
	apiv1 "github.com/xpropagation/xpropagator/api/v1/gen"
	"github.com/xpropagation/xpropagator/internal/config"
	"github.com/xpropagation/xpropagator/internal/core/gc"
	"github.com/xpropagation/xpropagator/internal/dllcore"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewPropagatorService(cfg *config.Config, logger *zap.Logger, satGC *gc.GC) *PropagationService {
	logger.Info("PropagatorService initialized")
	return &PropagationService{
		cfg:    cfg,
		logger: logger,
		gc:     satGC,
	}
}

func UtcToDS50(t time.Time) float64 {
	t = t.UTC()
	ref := time.Date(1950, 1, 1, 12, 0, 0, 0, time.UTC)
	return t.Sub(ref).Seconds() / 86400.0
}

func DS50ToUtc(ds50 float64) time.Time {
	ref := time.Date(1950, 1, 1, 12, 0, 0, 0, time.UTC)
	return ref.Add(time.Duration(ds50 * 86400.0 * float64(time.Second)))
}

func (propSrv *PropagationService) Ephem(req *apiv1.EphemRequest, srv apiv1.Propagator_EphemServer) error {
	if err := validateAnalytEphemRequest(req); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	globMu.Lock()
	defer globMu.Unlock()

	startTime := time.Now()
	ctx := srv.Context()

	resultsCh, errCh, senderDoneCh := propSrv.startResultSender(srv, propSrv.cfg.StreamChunkSize)

	for taskIdx, task := range req.GetTasks() {
		select {
		case <-ctx.Done():
			close(resultsCh)
			<-senderDoneCh
			return ctx.Err()
		default:
		}

		satKey, release, err := propSrv.gc.Acquire(ctx, task.GetSat().GetTleLn1(), task.GetSat().GetTleLn2())
		if err != nil {
			close(resultsCh)
			<-senderDoneCh
			return status.Errorf(codes.Internal, "failed to acquire satellite: %v", err)
		}

		err = propSrv.processTask(ctx, req, task, taskIdx, satKey, resultsCh, propSrv.cfg.StreamChunkSize)
		if release != nil {
			release()
		}

		if err != nil {
			close(resultsCh)
			<-senderDoneCh
			select {
			case err2 := <-errCh:
				if err2 != nil {
					return err2
				}
			default:
			}
			return err
		}
	}

	close(resultsCh)
	<-senderDoneCh

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	default:
	}

	propSrv.logger.Info("ephemeris data generation done", zap.Duration("time took", time.Since(startTime)))
	return nil
}

func (propSrv *PropagationService) startResultSender(
	srv apiv1.Propagator_EphemServer,
	bufSize int,
) (chan *apiv1.EphemResponse, chan error, chan struct{}) {
	resultsCh := make(chan *apiv1.EphemResponse, bufSize)
	errCh := make(chan error, 1)
	doneCh := make(chan struct{}, 1)

	go func() {
		defer close(doneCh)
		for res := range resultsCh {
			if err := srv.Send(res); err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
		}
	}()
	return resultsCh, errCh, doneCh
}

func (propSrv *PropagationService) processTask(
	ctx context.Context,
	req *apiv1.EphemRequest,
	task *apiv1.EphemTask,
	taskIdx int,
	satKey int64,
	resultsCh chan<- *apiv1.EphemResponse,
	chunkSize int,
) error {
	return propSrv.runAnalytGenEphems(ctx, req, task, taskIdx, satKey, resultsCh, chunkSize)
}

func resolveTimeGrid(task *apiv1.EphemTask, req *apiv1.EphemRequest) *apiv1.EphemTimeGrid {
	if task.GetTimeGrid() != nil {
		return task.GetTimeGrid()
	}
	return req.GetCommonTimeGrid()
}

func isDynamicTimeStep(grid *apiv1.EphemTimeGrid) bool {
	t, ok := grid.TimeStepType.(*apiv1.EphemTimeGrid_DynamicTimeStep)
	return ok && t.DynamicTimeStep
}

func getKnownTimeStep(grid *apiv1.EphemTimeGrid) float64 {
	t, ok := grid.TimeStepType.(*apiv1.EphemTimeGrid_KnownTimeStepPeriod)
	if ok {

		dur, err := duration.Parse(t.KnownTimeStepPeriod)
		if err != nil {
			panic(err)
		}

		step := dur.ToTimeDuration().Minutes()

		return step
	}

	t2, ok2 := grid.TimeStepType.(*apiv1.EphemTimeGrid_KnownTimeStepDs50)
	if ok2 {
		return t2.KnownTimeStepDs50 * 1440
	}
	return 0
}

func (propSrv *PropagationService) runAnalytGenEphems(ctx context.Context, req *apiv1.EphemRequest, task *apiv1.EphemTask, taskIdx int, satKey int64, resultsCh chan<- *apiv1.EphemResponse, chunkSize int) error {
	timeGrid := resolveTimeGrid(task, req)

	if timeGrid.TimeStartUtc != nil {
		timeGrid.TimeStartDs50 = UtcToDS50(timeGrid.TimeStartUtc.AsTime())
	}
	if timeGrid.TimeEndUtc != nil {
		timeGrid.TimeEndDs50 = UtcToDS50(timeGrid.TimeEndUtc.AsTime())
	}

	var timeStep float64
	if isDynamicTimeStep(timeGrid) {
		timeStep = -1
	} else {
		timeStep = getKnownTimeStep(timeGrid)
	}

	chunkID := 0
	timeStart := timeGrid.TimeStartDs50

	for {
		flat, n, next, done, errCode := dllcore.Sgp4GenEphems(satKey, timeStart, timeGrid.TimeEndDs50, timeStep, dllcore.EphemType(req.GetEphemType()), int32(chunkSize))
		if errCode == -10 {
			return fmt.Errorf("failed to generate ephemeris data: failed to allocate result buffer")
		}
		if errCode != 0 && n == 0 {
			if msg := getDllLastError(); msg != "" {
				return fmt.Errorf("failed to generate ephemeris data: %s", msg)
			}
		}

		if n > 0 {
			ephemResp, err := buildEphemResponse(req, task, taskIdx, flat, n, chunkID)
			if err != nil {
				return err
			}
			select {
			case resultsCh <- ephemResp:
				chunkID++
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if done {
			break
		}
		timeStart = next
	}
	return nil
}

func flatEphemsToEphemDataArr(flatEphems []float64) ([]*apiv1.EphemerisData, error) {
	if len(flatEphems) < 7 {
		return nil, fmt.Errorf("flat ephemerides array must have at least 7 elements")
	}
	if len(flatEphems)%7 != 0 {
		return nil, fmt.Errorf("flat ephemerides array length must be a multiple of 7")
	}

	count := len(flatEphems) / 7
	ephems := make([]*apiv1.EphemerisData, count)

	for i := 0; i < count; i++ {
		base := i * 7
		ephems[i] = &apiv1.EphemerisData{
			Ds50Time: flatEphems[base],
			X:        flatEphems[base+1],
			Y:        flatEphems[base+2],
			Z:        flatEphems[base+3],
			Vx:       flatEphems[base+4],
			Vy:       flatEphems[base+5],
			Vz:       flatEphems[base+6],
		}
	}

	return ephems, nil
}

func validateAnalytEphemRequest(req *apiv1.EphemRequest) error {
	validateGrid := func(grid *apiv1.EphemTimeGrid) error {
		if grid.GetTimeStartUtc() != nil && grid.GetTimeStartDs50() != 0 {
			return fmt.Errorf("invalid grid: start time cannot be given in UTC50, a start time in UTC is already specified. Please use only one format")
		}
		if grid.GetTimeEndUtc() != nil && grid.GetTimeEndDs50() != 0 {
			return fmt.Errorf("invalid configuration: end time cannot be given in UTC50, an end time in UTC is already specified. Please use only one format")
		}

		if grid.GetKnownTimeStepPeriod() != "" && grid.GetKnownTimeStepDs50() != 0 {
			return fmt.Errorf("invalid configuration: time step cannot be given in UTC50, a time step in UTC is already specified. Please use only one format")
		}
		return nil
	}

	if len(req.Tasks) == 0 {
		return fmt.Errorf("request must have at least one task")
	}

	if req.GetEphemType() != apiv1.EphemType(dllcore.EciEphemType) && req.GetEphemType() != apiv1.EphemType(dllcore.J2KEphemType) {
		return fmt.Errorf("invalid ephemerides type: %v (valid types: ECI, J2K)", req.GetEphemType())
	}

	if req.GetCommonTimeGrid() != nil {
		if err := validateGrid(req.GetCommonTimeGrid()); err != nil {
			return err
		}
	}

	for i, task := range req.GetTasks() {
		if err := validateSatellite(task.GetSat()); err != nil {
			return fmt.Errorf("task %d: %w", i, err)
		}
		if req.GetCommonTimeGrid() == nil && task.GetTimeGrid() == nil {
			return fmt.Errorf("task %d must have its own time grid since no common time grid is specified", i)
		}
		if task.GetTimeGrid() != nil {
			if err := validateGrid(task.GetTimeGrid()); err != nil {
				return fmt.Errorf("task %d: %w", i, err)
			}
		}
	}

	return nil
}

func buildEphemResponse(req *apiv1.EphemRequest, task *apiv1.EphemTask, taskIdx int, flat []float64, count int, chunkID int) (*apiv1.EphemResponse, error) {
	ephemDataArr, err := flatEphemsToEphemDataArr(flat)
	if err != nil {
		return nil, err
	}
	return &apiv1.EphemResponse{
		ReqId:         req.GetReqId(),
		StreamId:      int64(taskIdx),
		StreamChunkId: int64(chunkID),
		Result: &apiv1.EphemOut{
			TaskId:           task.GetTaskId(),
			EphemData:        ephemDataArr,
			EphemPointsCount: int64(count),
		},
	}, nil
}
