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

// Package core integration tests for the Propagator gRPC API.
// These tests require the SGP4 DLL libraries to be present.
//
// Run with: go test -tags=integration -v ./internal/core/...

package core

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	apiv1 "github.com/xpropagation/xpropagator/api/v1/gen"
	"github.com/xpropagation/xpropagator/internal/config"
	"github.com/xpropagation/xpropagator/internal/core/gc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const bufSize = 1024 * 1024

// Test TLE data for ISS
const (
	testTLELine1 = "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042"
	testTLELine2 = "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506"
	testNoradID  = 25544
	testSatName  = "ISS (ZARYA)"
)

// Test TLE data for second satellite
const (
	testTLELine1_2 = "1 65271U 25183A   25282.36302114 0.00010000  00000-0  55866-4 0    07"
	testTLELine2_2 = "2 65271  48.7951   8.5514 0002000  85.4867 277.3551 15.78566782    05"
	testNoradID_2  = 65271
	testSatName_2  = "X-37B OTV-8"
)

// testServer holds the test server infrastructure
type testServer struct {
	listener *bufconn.Listener
	server   *grpc.Server
	service  *PropagationService
	gc       *gc.GC
}

// newTestServer creates a new test server with the propagation service
func newTestServer(t *testing.T) *testServer {
	t.Helper()

	// Create test configuration
	cfg := &config.Config{
		Reflection:             false,
		StreamChunkSize:        10,
		GracefulStopTimeoutSec: 10 * time.Second,
		GC: &config.GCConfig{
			MaxLoadedSatsGc:    100,
			IdleTTLGcMin:       10 * time.Minute,
			SweepIntervalGcMin: 1 * time.Minute,
		},
		TLS: &config.TLSConfig{
			Enabled: false,
		},
	}

	// Create logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create GC
	satGC := gc.NewGC(
		cfg.GC.MaxLoadedSatsGc,
		cfg.GC.IdleTTLGcMin,
		cfg.GC.SweepIntervalGcMin,
	)

	// Create service
	service := NewPropagatorService(cfg, logger, satGC)

	// Create gRPC server
	listener := bufconn.Listen(bufSize)
	server := grpc.NewServer()
	apiv1.RegisterPropagatorServer(server, service)

	// Start server in background
	go func() {
		if err := server.Serve(listener); err != nil {
			// Server stopped
		}
	}()

	return &testServer{
		listener: listener,
		server:   server,
		service:  service,
		gc:       satGC,
	}
}

// close stops the test server and cleans up resources
func (ts *testServer) close() {
	ts.server.Stop()
	ts.gc.Close()
	ts.listener.Close()
}

// dial creates a client connection to the test server
func (ts *testServer) dial(ctx context.Context) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return ts.listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

// =============================================================================
// Info API Tests
// =============================================================================

func TestAPI_Info(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	resp, err := client.Info(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("Info() failed: %v", err)
	}

	// Verify response fields
	if resp.GetName() == "" {
		t.Error("Expected non-empty service name")
	}

	if resp.GetTimestamp() == nil {
		t.Error("Expected non-nil timestamp")
	}

	// Check that timestamp is recent (within last minute)
	ts_time := resp.GetTimestamp().AsTime()
	if time.Since(ts_time) > time.Minute {
		t.Errorf("Timestamp too old: %v", ts_time)
	}

	t.Logf("Info response: Name=%s, Version=%s, AstroLib=%s, SGP4Lib=%s",
		resp.GetName(), resp.GetVersion(), resp.GetAstroStdLibInfo(), resp.GetSgp4LibInfo())
}

// =============================================================================
// Prop API Tests
// =============================================================================

func TestAPI_Prop_DS50Time(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	req := &apiv1.PropRequest{
		ReqId:    1,
		TimeType: apiv1.TimeType_TimeDs50,
		Task: &apiv1.PropTask{
			Time: 27744.5, // DS50 time
			Sat: &apiv1.Satellite{
				NoradId: testNoradID,
				Name:    testSatName,
				TleLn1:  testTLELine1,
				TleLn2:  testTLELine2,
			},
		},
	}

	resp, err := client.Prop(ctx, req)
	if err != nil {
		t.Fatalf("Prop() failed: %v", err)
	}

	// Verify response
	if resp.GetReqId() != 1 {
		t.Errorf("ReqId = %d, want 1", resp.GetReqId())
	}

	result := resp.GetResult()
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check that we got valid position/velocity data
	if result.GetDs50Time() == 0 {
		t.Error("Expected non-zero DS50 time")
	}

	t.Logf("Prop result: DS50=%f, Pos=[%f, %f, %f], Vel=[%f, %f, %f]",
		result.GetDs50Time(),
		result.GetX(), result.GetY(), result.GetZ(),
		result.GetVx(), result.GetVy(), result.GetVz())
}

func TestAPI_Prop_UTCTime(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	propTime := time.Date(2025, 12, 18, 12, 0, 0, 0, time.UTC)

	req := &apiv1.PropRequest{
		ReqId:    2,
		TimeType: apiv1.TimeType_TimeDs50,
		Task: &apiv1.PropTask{
			TimeUtc: timestamppb.New(propTime),
			Sat: &apiv1.Satellite{
				NoradId: testNoradID,
				Name:    testSatName,
				TleLn1:  testTLELine1,
				TleLn2:  testTLELine2,
			},
		},
	}

	resp, err := client.Prop(ctx, req)
	if err != nil {
		t.Fatalf("Prop() with UTC time failed: %v", err)
	}

	if resp.GetReqId() != 2 {
		t.Errorf("ReqId = %d, want 2", resp.GetReqId())
	}

	result := resp.GetResult()
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	t.Logf("Prop UTC result: DS50=%f, MSE=%f", result.GetDs50Time(), result.GetMseTime())
}

func TestAPI_Prop_MSETime(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	req := &apiv1.PropRequest{
		ReqId:    3,
		TimeType: apiv1.TimeType_TimeMse,
		Task: &apiv1.PropTask{
			Time: 1440.0, // 1440 minutes = 1 day from TLE epoch
			Sat: &apiv1.Satellite{
				NoradId: testNoradID,
				Name:    testSatName,
				TleLn1:  testTLELine1,
				TleLn2:  testTLELine2,
			},
		},
	}

	resp, err := client.Prop(ctx, req)
	if err != nil {
		t.Fatalf("Prop() with MSE time failed: %v", err)
	}

	if resp.GetReqId() != 3 {
		t.Errorf("ReqId = %d, want 3", resp.GetReqId())
	}

	result := resp.GetResult()
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.GetMseTime() == 0 {
		t.Error("Expected non-zero MSE time in result")
	}

	t.Logf("Prop MSE result: DS50=%f, MSE=%f", result.GetDs50Time(), result.GetMseTime())
}

func TestAPI_Prop_MultipleSatellites(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	satellites := []struct {
		noradID int64
		name    string
		tle1    string
		tle2    string
	}{
		{testNoradID, testSatName, testTLELine1, testTLELine2},
		{testNoradID_2, testSatName_2, testTLELine1_2, testTLELine2_2},
	}

	for i, sat := range satellites {
		req := &apiv1.PropRequest{
			ReqId:    int64(i + 1),
			TimeType: apiv1.TimeType_TimeDs50,
			Task: &apiv1.PropTask{
				Time: 27744.5,
				Sat: &apiv1.Satellite{
					NoradId: sat.noradID,
					Name:    sat.name,
					TleLn1:  sat.tle1,
					TleLn2:  sat.tle2,
				},
			},
		}

		resp, err := client.Prop(ctx, req)
		if err != nil {
			t.Errorf("Prop() for satellite %s failed: %v", sat.name, err)
			continue
		}

		if resp.GetResult() == nil {
			t.Errorf("Expected non-nil result for satellite %s", sat.name)
		}

		t.Logf("Satellite %s propagated successfully", sat.name)
	}
}

// =============================================================================
// Ephem API Tests
// =============================================================================

func TestAPI_Ephem_ECI_SingleSatellite(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	startTime := time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 12, 18, 1, 0, 0, 0, time.UTC) // 1 hour

	req := &apiv1.EphemRequest{
		ReqId:     1,
		EphemType: apiv1.EphemType_EphemEci,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartUtc: timestamppb.New(startTime),
			TimeEndUtc:   timestamppb.New(endTime),
			TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepPeriod{
				KnownTimeStepPeriod: "PT10M", // 10 minute intervals
			},
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				Sat: &apiv1.Satellite{
					NoradId: testNoradID,
					Name:    testSatName,
					TleLn1:  testTLELine1,
					TleLn2:  testTLELine2,
				},
			},
		},
	}

	stream, err := client.Ephem(ctx, req)
	if err != nil {
		t.Fatalf("Ephem() failed: %v", err)
	}

	var totalPoints int64
	var chunkCount int

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Stream recv failed: %v", err)
		}

		chunkCount++
		result := resp.GetResult()
		if result == nil {
			t.Error("Expected non-nil result in stream")
			continue
		}

		totalPoints += result.GetEphemPointsCount()

		// Verify ephemeris data
		for _, ephem := range result.GetEphemData() {
			if ephem.GetDs50Time() == 0 {
				t.Error("Expected non-zero DS50 time in ephemeris")
			}
		}

		t.Logf("Received chunk %d with %d points", resp.GetStreamChunkId(), result.GetEphemPointsCount())
	}

	if totalPoints == 0 {
		t.Error("Expected at least one ephemeris point")
	}

	t.Logf("Total: %d chunks, %d ephemeris points", chunkCount, totalPoints)
}

func TestAPI_Ephem_J2K_SingleSatellite(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	req := &apiv1.EphemRequest{
		ReqId:     2,
		EphemType: apiv1.EphemType_EphemJ2K,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartDs50: 27744.0,
			TimeEndDs50:   27744.1, // ~2.4 hours
			TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepDs50{
				KnownTimeStepDs50: 0.01, // ~14.4 minutes
			},
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				Sat: &apiv1.Satellite{
					NoradId: testNoradID,
					Name:    testSatName,
					TleLn1:  testTLELine1,
					TleLn2:  testTLELine2,
				},
			},
		},
	}

	stream, err := client.Ephem(ctx, req)
	if err != nil {
		t.Fatalf("Ephem() J2K failed: %v", err)
	}

	var totalPoints int64

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Stream recv failed: %v", err)
		}

		result := resp.GetResult()
		if result != nil {
			totalPoints += result.GetEphemPointsCount()
		}
	}

	if totalPoints == 0 {
		t.Error("Expected at least one J2K ephemeris point")
	}

	t.Logf("J2K ephemeris: %d total points", totalPoints)
}

func TestAPI_Ephem_MultipleSatellites(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	startTime := time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 12, 18, 0, 30, 0, 0, time.UTC) // 30 minutes

	req := &apiv1.EphemRequest{
		ReqId:     3,
		EphemType: apiv1.EphemType_EphemEci,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartUtc: timestamppb.New(startTime),
			TimeEndUtc:   timestamppb.New(endTime),
			TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepPeriod{
				KnownTimeStepPeriod: "PT5M", // 5 minute intervals
			},
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				Sat: &apiv1.Satellite{
					NoradId: testNoradID,
					Name:    testSatName,
					TleLn1:  testTLELine1,
					TleLn2:  testTLELine2,
				},
			},
			{
				TaskId: 2,
				Sat: &apiv1.Satellite{
					NoradId: testNoradID_2,
					Name:    testSatName_2,
					TleLn1:  testTLELine1_2,
					TleLn2:  testTLELine2_2,
				},
			},
		},
	}

	stream, err := client.Ephem(ctx, req)
	if err != nil {
		t.Fatalf("Ephem() multiple satellites failed: %v", err)
	}

	taskPoints := make(map[int64]int64)

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Stream recv failed: %v", err)
		}

		result := resp.GetResult()
		if result != nil {
			taskPoints[result.GetTaskId()] += result.GetEphemPointsCount()
		}
	}

	// Verify we got data for both satellites
	if len(taskPoints) != 2 {
		t.Errorf("Expected data for 2 satellites, got %d", len(taskPoints))
	}

	for taskID, points := range taskPoints {
		if points == 0 {
			t.Errorf("Task %d has 0 ephemeris points", taskID)
		}
		t.Logf("Task %d: %d ephemeris points", taskID, points)
	}
}

func TestAPI_Ephem_IndividualTimeGrids(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	req := &apiv1.EphemRequest{
		ReqId:          4,
		EphemType:      apiv1.EphemType_EphemEci,
		CommonTimeGrid: nil, // No common grid
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				TimeGrid: &apiv1.EphemTimeGrid{
					TimeStartDs50: 27744.0,
					TimeEndDs50:   27744.05,
					TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepDs50{
						KnownTimeStepDs50: 0.01,
					},
				},
				Sat: &apiv1.Satellite{
					NoradId: testNoradID,
					Name:    testSatName,
					TleLn1:  testTLELine1,
					TleLn2:  testTLELine2,
				},
			},
			{
				TaskId: 2,
				TimeGrid: &apiv1.EphemTimeGrid{
					TimeStartDs50: 27745.0, // Different time range
					TimeEndDs50:   27745.05,
					TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepDs50{
						KnownTimeStepDs50: 0.01,
					},
				},
				Sat: &apiv1.Satellite{
					NoradId: testNoradID_2,
					Name:    testSatName_2,
					TleLn1:  testTLELine1_2,
					TleLn2:  testTLELine2_2,
				},
			},
		},
	}

	stream, err := client.Ephem(ctx, req)
	if err != nil {
		t.Fatalf("Ephem() with individual time grids failed: %v", err)
	}

	taskPoints := make(map[int64]int64)

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Stream recv failed: %v", err)
		}

		result := resp.GetResult()
		if result != nil {
			taskPoints[result.GetTaskId()] += result.GetEphemPointsCount()
		}
	}

	if len(taskPoints) != 2 {
		t.Errorf("Expected data for 2 tasks, got %d", len(taskPoints))
	}

	t.Logf("Individual time grids: Task 1=%d points, Task 2=%d points",
		taskPoints[1], taskPoints[2])
}

func TestAPI_Ephem_DynamicTimeStep(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	req := &apiv1.EphemRequest{
		ReqId:     10,
		EphemType: apiv1.EphemType_EphemEci,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartDs50: 27744.0,
			TimeEndDs50:   27744.05,
			TimeStepType: &apiv1.EphemTimeGrid_DynamicTimeStep{
				DynamicTimeStep: true,
			},
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				Sat: &apiv1.Satellite{
					NoradId: testNoradID,
					Name:    testSatName,
					TleLn1:  testTLELine1,
					TleLn2:  testTLELine2,
				},
			},
		},
	}

	stream, err := client.Ephem(ctx, req)
	if err != nil {
		t.Fatalf("Ephem() with dynamic time step failed: %v", err)
	}

	var totalPoints int64

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Stream recv failed: %v", err)
		}

		result := resp.GetResult()
		if result != nil {
			totalPoints += result.GetEphemPointsCount()
		}
	}

	t.Logf("Dynamic time step: %d total points", totalPoints)
}

func TestAPI_Ephem_VariousISO8601Durations(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	durations := []struct {
		name     string
		duration string
	}{
		{"1 minute", "PT1M"},
		{"5 minutes", "PT5M"},
		{"30 seconds", "PT30S"},
		{"1.5 minutes", "PT1.5M"},
		{"1 hour", "PT1H"},
		{"90 seconds", "PT90S"},
	}

	startTime := time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 12, 18, 0, 10, 0, 0, time.UTC) // 10 minutes

	for i, d := range durations {
		t.Run(d.name, func(t *testing.T) {
			req := &apiv1.EphemRequest{
				ReqId:     int64(20 + i),
				EphemType: apiv1.EphemType_EphemEci,
				CommonTimeGrid: &apiv1.EphemTimeGrid{
					TimeStartUtc: timestamppb.New(startTime),
					TimeEndUtc:   timestamppb.New(endTime),
					TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepPeriod{
						KnownTimeStepPeriod: d.duration,
					},
				},
				Tasks: []*apiv1.EphemTask{
					{
						TaskId: 1,
						Sat: &apiv1.Satellite{
							NoradId: testNoradID,
							Name:    testSatName,
							TleLn1:  testTLELine1,
							TleLn2:  testTLELine2,
						},
					},
				},
			}

			stream, err := client.Ephem(ctx, req)
			if err != nil {
				t.Fatalf("Ephem() with duration %s failed: %v", d.duration, err)
			}

			var totalPoints int64
			for {
				resp, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("Stream recv failed: %v", err)
				}
				if resp.GetResult() != nil {
					totalPoints += resp.GetResult().GetEphemPointsCount()
				}
			}

			t.Logf("Duration %s: %d points", d.duration, totalPoints)
		})
	}
}

func TestAPI_Ephem_LongTimeRange_FullDay(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	startTime := time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 12, 19, 0, 0, 0, 0, time.UTC) // Full 24 hours

	req := &apiv1.EphemRequest{
		ReqId:     30,
		EphemType: apiv1.EphemType_EphemEci,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartUtc: timestamppb.New(startTime),
			TimeEndUtc:   timestamppb.New(endTime),
			TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepPeriod{
				KnownTimeStepPeriod: "PT10M", // 10 minute intervals = 144 points
			},
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				Sat: &apiv1.Satellite{
					NoradId: testNoradID,
					Name:    testSatName,
					TleLn1:  testTLELine1,
					TleLn2:  testTLELine2,
				},
			},
		},
	}

	stream, err := client.Ephem(ctx, req)
	if err != nil {
		t.Fatalf("Ephem() for full day failed: %v", err)
	}

	var totalPoints int64
	var chunkCount int

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Stream recv failed: %v", err)
		}

		chunkCount++
		if resp.GetResult() != nil {
			totalPoints += resp.GetResult().GetEphemPointsCount()
		}
	}

	// 24 hours / 10 minutes = 144 points expected (approximately)
	if totalPoints < 100 {
		t.Errorf("Expected at least 100 points for full day, got %d", totalPoints)
	}

	t.Logf("Full day ephemeris: %d chunks, %d total points", chunkCount, totalPoints)
}

func TestAPI_Ephem_HighFrequency(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	startTime := time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 12, 18, 0, 30, 0, 0, time.UTC) // 30 minutes

	req := &apiv1.EphemRequest{
		ReqId:     31,
		EphemType: apiv1.EphemType_EphemEci,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartUtc: timestamppb.New(startTime),
			TimeEndUtc:   timestamppb.New(endTime),
			TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepPeriod{
				KnownTimeStepPeriod: "PT1M", // 1 minute intervals = 30 points
			},
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				Sat: &apiv1.Satellite{
					NoradId: testNoradID,
					Name:    testSatName,
					TleLn1:  testTLELine1,
					TleLn2:  testTLELine2,
				},
			},
		},
	}

	stream, err := client.Ephem(ctx, req)
	if err != nil {
		t.Fatalf("Ephem() high frequency failed: %v", err)
	}

	var totalPoints int64

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Stream recv failed: %v", err)
		}

		if resp.GetResult() != nil {
			totalPoints += resp.GetResult().GetEphemPointsCount()
		}
	}

	// 30 minutes / 1 minute = ~30 points expected
	if totalPoints < 25 || totalPoints > 35 {
		t.Errorf("Expected around 30 points for 30 minutes at 1-min intervals, got %d", totalPoints)
	}

	t.Logf("High frequency ephemeris: %d total points", totalPoints)
}

func TestAPI_Ephem_StreamMetadataVerification(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	req := &apiv1.EphemRequest{
		ReqId:     40,
		EphemType: apiv1.EphemType_EphemEci,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartDs50: 27744.0,
			TimeEndDs50:   27744.1,
			TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepDs50{
				KnownTimeStepDs50: 0.001, // Small step = many points = multiple chunks
			},
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 100,
				Sat: &apiv1.Satellite{
					NoradId: testNoradID,
					Name:    testSatName,
					TleLn1:  testTLELine1,
					TleLn2:  testTLELine2,
				},
			},
		},
	}

	stream, err := client.Ephem(ctx, req)
	if err != nil {
		t.Fatalf("Ephem() failed: %v", err)
	}

	var lastChunkID int64 = -1
	var responses []*apiv1.EphemResponse

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Stream recv failed: %v", err)
		}

		responses = append(responses, resp)

		// Verify ReqId matches
		if resp.GetReqId() != 40 {
			t.Errorf("ReqId = %d, want 40", resp.GetReqId())
		}

		// Verify TaskId matches
		if resp.GetResult() != nil && resp.GetResult().GetTaskId() != 100 {
			t.Errorf("TaskId = %d, want 100", resp.GetResult().GetTaskId())
		}

		// Verify chunk IDs are sequential
		currentChunkID := resp.GetStreamChunkId()
		if lastChunkID >= 0 && currentChunkID != lastChunkID+1 {
			t.Errorf("Chunk IDs not sequential: got %d after %d", currentChunkID, lastChunkID)
		}
		lastChunkID = currentChunkID

		// Verify ephemeris count matches actual data
		if resp.GetResult() != nil {
			declaredCount := resp.GetResult().GetEphemPointsCount()
			actualCount := int64(len(resp.GetResult().GetEphemData()))
			if declaredCount != actualCount {
				t.Errorf("EphemPointsCount=%d but actual data has %d points", declaredCount, actualCount)
			}
		}
	}

	if len(responses) == 0 {
		t.Error("Expected at least one response")
	}

	t.Logf("Stream metadata verified: %d responses, chunk IDs 0-%d", len(responses), lastChunkID)
}

func TestAPI_Ephem_ManySatellites(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	// Use same TLE but different task IDs to simulate multiple satellites
	tasks := make([]*apiv1.EphemTask, 5)
	for i := 0; i < 5; i++ {
		var tle1, tle2 string
		var noradID int64
		if i%2 == 0 {
			tle1, tle2, noradID = testTLELine1, testTLELine2, testNoradID
		} else {
			tle1, tle2, noradID = testTLELine1_2, testTLELine2_2, testNoradID_2
		}
		tasks[i] = &apiv1.EphemTask{
			TaskId: int64(i + 1),
			Sat: &apiv1.Satellite{
				NoradId: noradID,
				Name:    "Satellite " + string(rune('A'+i)),
				TleLn1:  tle1,
				TleLn2:  tle2,
			},
		}
	}

	req := &apiv1.EphemRequest{
		ReqId:     50,
		EphemType: apiv1.EphemType_EphemEci,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartDs50: 27744.0,
			TimeEndDs50:   27744.02,
			TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepDs50{
				KnownTimeStepDs50: 0.005,
			},
		},
		Tasks: tasks,
	}

	stream, err := client.Ephem(ctx, req)
	if err != nil {
		t.Fatalf("Ephem() with many satellites failed: %v", err)
	}

	taskPoints := make(map[int64]int64)

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Stream recv failed: %v", err)
		}

		result := resp.GetResult()
		if result != nil {
			taskPoints[result.GetTaskId()] += result.GetEphemPointsCount()
		}
	}

	// Verify we got data for all 5 tasks
	if len(taskPoints) != 5 {
		t.Errorf("Expected data for 5 tasks, got %d", len(taskPoints))
	}

	for taskID := int64(1); taskID <= 5; taskID++ {
		points := taskPoints[taskID]
		if points == 0 {
			t.Errorf("Task %d has 0 points", taskID)
		}
		t.Logf("Task %d: %d points", taskID, points)
	}
}

func TestAPI_Ephem_ShortTimeRange_SingleOrbit(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	// ISS orbital period is about 92 minutes
	startTime := time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 12, 18, 1, 32, 0, 0, time.UTC) // ~92 minutes

	req := &apiv1.EphemRequest{
		ReqId:     60,
		EphemType: apiv1.EphemType_EphemEci,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartUtc: timestamppb.New(startTime),
			TimeEndUtc:   timestamppb.New(endTime),
			TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepPeriod{
				KnownTimeStepPeriod: "PT1M", // 1 minute = ~92 points
			},
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				Sat: &apiv1.Satellite{
					NoradId: testNoradID,
					Name:    testSatName,
					TleLn1:  testTLELine1,
					TleLn2:  testTLELine2,
				},
			},
		},
	}

	stream, err := client.Ephem(ctx, req)
	if err != nil {
		t.Fatalf("Ephem() for single orbit failed: %v", err)
	}

	var totalPoints int64
	var firstDS50, lastDS50 float64

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Stream recv failed: %v", err)
		}

		result := resp.GetResult()
		if result != nil {
			totalPoints += result.GetEphemPointsCount()
			for _, ephem := range result.GetEphemData() {
				if firstDS50 == 0 {
					firstDS50 = ephem.GetDs50Time()
				}
				lastDS50 = ephem.GetDs50Time()
			}
		}
	}

	// Calculate time span
	timeSpanDays := lastDS50 - firstDS50
	timeSpanMinutes := timeSpanDays * 1440

	t.Logf("Single orbit: %d points, time span=%.2f minutes", totalPoints, timeSpanMinutes)
}

func TestAPI_Ephem_MixedTimeGrids(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	// Task 1 uses common time grid, Task 2 has its own
	req := &apiv1.EphemRequest{
		ReqId:     80,
		EphemType: apiv1.EphemType_EphemJ2K,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartDs50: 27744.0,
			TimeEndDs50:   27744.02,
			TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepDs50{
				KnownTimeStepDs50: 0.005,
			},
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId:   1,
				TimeGrid: nil, // Uses common grid
				Sat: &apiv1.Satellite{
					NoradId: testNoradID,
					Name:    testSatName,
					TleLn1:  testTLELine1,
					TleLn2:  testTLELine2,
				},
			},
			{
				TaskId: 2,
				TimeGrid: &apiv1.EphemTimeGrid{
					TimeStartDs50: 27745.0, // Different range
					TimeEndDs50:   27745.02,
					TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepDs50{
						KnownTimeStepDs50: 0.01, // Different step
					},
				},
				Sat: &apiv1.Satellite{
					NoradId: testNoradID_2,
					Name:    testSatName_2,
					TleLn1:  testTLELine1_2,
					TleLn2:  testTLELine2_2,
				},
			},
		},
	}

	stream, err := client.Ephem(ctx, req)
	if err != nil {
		t.Fatalf("Ephem() with mixed time grids failed: %v", err)
	}

	taskPoints := make(map[int64]int64)
	taskDS50Ranges := make(map[int64][2]float64) // min, max DS50

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Stream recv failed: %v", err)
		}

		result := resp.GetResult()
		if result == nil {
			continue
		}

		taskID := result.GetTaskId()
		taskPoints[taskID] += result.GetEphemPointsCount()

		for _, ephem := range result.GetEphemData() {
			ds50 := ephem.GetDs50Time()
			if _, ok := taskDS50Ranges[taskID]; !ok {
				taskDS50Ranges[taskID] = [2]float64{ds50, ds50}
			} else {
				if ds50 < taskDS50Ranges[taskID][0] {
					taskDS50Ranges[taskID] = [2]float64{ds50, taskDS50Ranges[taskID][1]}
				}
				if ds50 > taskDS50Ranges[taskID][1] {
					taskDS50Ranges[taskID] = [2]float64{taskDS50Ranges[taskID][0], ds50}
				}
			}
		}
	}

	// Verify both tasks have data
	if len(taskPoints) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(taskPoints))
	}

	// Verify tasks have different time ranges
	t.Logf("Task 1: %d points, DS50 range [%f, %f]",
		taskPoints[1], taskDS50Ranges[1][0], taskDS50Ranges[1][1])
	t.Logf("Task 2: %d points, DS50 range [%f, %f]",
		taskPoints[2], taskDS50Ranges[2][0], taskDS50Ranges[2][1])

	// Task 1 should be around 27744, Task 2 around 27745
	if taskDS50Ranges[1][0] > 27745 || taskDS50Ranges[2][0] < 27745 {
		t.Error("Tasks appear to have wrong time ranges")
	}
}

func TestAPI_Ephem_ContextCancellation(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	// Short timeout to trigger cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		// Connection might fail due to timeout
		t.Logf("Connection failed (expected with short timeout): %v", err)
		return
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	// Large request that should take a while
	req := &apiv1.EphemRequest{
		ReqId:     90,
		EphemType: apiv1.EphemType_EphemEci,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartDs50: 27744.0,
			TimeEndDs50:   27754.0, // 10 days
			TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepDs50{
				KnownTimeStepDs50: 0.0001, // Very small step = many points
			},
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				Sat: &apiv1.Satellite{
					NoradId: testNoradID,
					Name:    testSatName,
					TleLn1:  testTLELine1,
					TleLn2:  testTLELine2,
				},
			},
		},
	}

	stream, err := client.Ephem(ctx, req)
	if err != nil {
		// Expected if timeout already triggered
		t.Logf("Ephem() failed (expected): %v", err)
		return
	}

	var receivedPoints int64
	for {
		resp, err := stream.Recv()
		if err != nil {
			// Expected: context deadline exceeded
			t.Logf("Stream ended with error (expected): %v", err)
			break
		}
		if resp.GetResult() != nil {
			receivedPoints += resp.GetResult().GetEphemPointsCount()
		}
	}

	t.Logf("Received %d points before cancellation", receivedPoints)
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestAPI_Prop_InvalidTLE(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	req := &apiv1.PropRequest{
		ReqId:    100,
		TimeType: apiv1.TimeType_TimeDs50,
		Task: &apiv1.PropTask{
			Time: 27744.5,
			Sat: &apiv1.Satellite{
				NoradId: 99999,
				Name:    "Invalid",
				TleLn1:  "1 XXXXX INVALID TLE LINE",
				TleLn2:  "2 XXXXX INVALID TLE LINE",
			},
		},
	}

	_, err = client.Prop(ctx, req)
	if err == nil {
		t.Error("Expected error for invalid TLE")
	}

	t.Logf("Prop with invalid TLE correctly returned error: %v", err)
}

func TestAPI_Prop_MissingTask(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	req := &apiv1.PropRequest{
		ReqId:    101,
		TimeType: apiv1.TimeType_TimeDs50,
		Task:     nil,
	}

	_, err = client.Prop(ctx, req)
	if err == nil {
		t.Error("Expected error for missing task")
	}

	t.Logf("Prop with missing task correctly returned error: %v", err)
}

func TestAPI_Ephem_NoTasks(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	req := &apiv1.EphemRequest{
		ReqId:     102,
		EphemType: apiv1.EphemType_EphemEci,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartDs50: 27744.0,
			TimeEndDs50:   27745.0,
		},
		Tasks: []*apiv1.EphemTask{}, // Empty tasks
	}

	stream, err := client.Ephem(ctx, req)
	if err != nil {
		// Some implementations may fail immediately
		t.Logf("Ephem with no tasks returned immediate error: %v", err)
		return
	}

	// Or fail on first recv
	_, err = stream.Recv()
	if err == nil || err == io.EOF {
		t.Error("Expected error for empty tasks, got success or EOF")
	}

	t.Logf("Ephem with no tasks correctly returned error: %v", err)
}

// =============================================================================
// Concurrency Tests
// =============================================================================

func TestAPI_ConcurrentPropRequests(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	const numRequests = 10
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(reqID int64) {
			req := &apiv1.PropRequest{
				ReqId:    reqID,
				TimeType: apiv1.TimeType_TimeDs50,
				Task: &apiv1.PropTask{
					Time: 27744.5 + float64(reqID)*0.01,
					Sat: &apiv1.Satellite{
						NoradId: testNoradID,
						Name:    testSatName,
						TleLn1:  testTLELine1,
						TleLn2:  testTLELine2,
					},
				},
			}

			resp, err := client.Prop(ctx, req)
			if err != nil {
				results <- err
				return
			}

			if resp.GetReqId() != reqID {
				results <- err
				return
			}

			results <- nil
		}(int64(i + 1))
	}

	var errors []error
	for i := 0; i < numRequests; i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		t.Errorf("Concurrent requests had %d errors: %v", len(errors), errors)
	} else {
		t.Logf("All %d concurrent Prop requests succeeded", numRequests)
	}
}

// ephemResult holds the result of a concurrent Ephem request
type ephemResult struct {
	reqID       int64
	totalPoints int64
	chunkCount  int
	taskPoints  map[int64]int64
	err         error
	startTime   time.Time
	endTime     time.Time
}

func TestAPI_ConcurrentEphemRequests(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	const numRequests = 5
	results := make(chan ephemResult, numRequests)

	startTime := time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 12, 18, 0, 30, 0, 0, time.UTC) // 30 minutes

	// Launch concurrent Ephem requests
	for i := 0; i < numRequests; i++ {
		go func(reqID int64) {
			result := ephemResult{
				reqID:      reqID,
				taskPoints: make(map[int64]int64),
				startTime:  time.Now(),
			}

			req := &apiv1.EphemRequest{
				ReqId:     reqID,
				EphemType: apiv1.EphemType_EphemEci,
				CommonTimeGrid: &apiv1.EphemTimeGrid{
					TimeStartUtc: timestamppb.New(startTime),
					TimeEndUtc:   timestamppb.New(endTime),
					TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepPeriod{
						KnownTimeStepPeriod: "PT5M", // 5 minute intervals
					},
				},
				Tasks: []*apiv1.EphemTask{
					{
						TaskId: reqID * 10, // Unique task ID per request
						Sat: &apiv1.Satellite{
							NoradId: testNoradID,
							Name:    testSatName,
							TleLn1:  testTLELine1,
							TleLn2:  testTLELine2,
						},
					},
				},
			}

			stream, err := client.Ephem(ctx, req)
			if err != nil {
				result.err = err
				result.endTime = time.Now()
				results <- result
				return
			}

			for {
				resp, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					result.err = err
					break
				}

				result.chunkCount++

				// Verify ReqId matches
				if resp.GetReqId() != reqID {
					result.err = io.ErrUnexpectedEOF // Use as marker for mismatch
					break
				}

				if resp.GetResult() != nil {
					taskID := resp.GetResult().GetTaskId()
					points := resp.GetResult().GetEphemPointsCount()
					result.taskPoints[taskID] += points
					result.totalPoints += points

					// Verify task ID is what we expect
					expectedTaskID := reqID * 10
					if taskID != expectedTaskID {
						result.err = io.ErrUnexpectedEOF // Use as marker for mismatch
						break
					}
				}
			}

			result.endTime = time.Now()
			results <- result
		}(int64(i + 1))
	}

	// Collect results
	var allResults []ephemResult
	for i := 0; i < numRequests; i++ {
		allResults = append(allResults, <-results)
	}

	// Analyze results
	var errors []error
	var totalDuration time.Duration
	for _, r := range allResults {
		duration := r.endTime.Sub(r.startTime)
		totalDuration += duration

		if r.err != nil {
			errors = append(errors, r.err)
			t.Errorf("Request %d failed: %v", r.reqID, r.err)
		} else {
			t.Logf("Request %d: %d points, %d chunks, duration=%v",
				r.reqID, r.totalPoints, r.chunkCount, duration)
		}
	}

	if len(errors) > 0 {
		t.Errorf("Concurrent Ephem requests had %d errors", len(errors))
	} else {
		t.Logf("All %d concurrent Ephem requests succeeded", numRequests)
		t.Logf("Total processing time: %v (with global lock serialization)", totalDuration)
	}
}

func TestAPI_ConcurrentEphemRequests_DifferentSatellites(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	const numRequests = 4
	results := make(chan ephemResult, numRequests)

	// Satellite configurations for different requests
	satellites := []struct {
		noradID int64
		name    string
		tle1    string
		tle2    string
	}{
		{testNoradID, testSatName, testTLELine1, testTLELine2},
		{testNoradID_2, testSatName_2, testTLELine1_2, testTLELine2_2},
		{testNoradID, testSatName, testTLELine1, testTLELine2},
		{testNoradID_2, testSatName_2, testTLELine1_2, testTLELine2_2},
	}

	// Launch concurrent Ephem requests with different satellites
	for i := 0; i < numRequests; i++ {
		go func(reqID int64, sat struct {
			noradID int64
			name    string
			tle1    string
			tle2    string
		}) {
			result := ephemResult{
				reqID:      reqID,
				taskPoints: make(map[int64]int64),
				startTime:  time.Now(),
			}

			req := &apiv1.EphemRequest{
				ReqId:     reqID,
				EphemType: apiv1.EphemType_EphemEci,
				CommonTimeGrid: &apiv1.EphemTimeGrid{
					TimeStartDs50: 27744.0,
					TimeEndDs50:   27744.05, // ~72 minutes
					TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepDs50{
						KnownTimeStepDs50: 0.01, // ~14.4 minutes
					},
				},
				Tasks: []*apiv1.EphemTask{
					{
						TaskId: reqID * 100,
						Sat: &apiv1.Satellite{
							NoradId: sat.noradID,
							Name:    sat.name,
							TleLn1:  sat.tle1,
							TleLn2:  sat.tle2,
						},
					},
				},
			}

			stream, err := client.Ephem(ctx, req)
			if err != nil {
				result.err = err
				result.endTime = time.Now()
				results <- result
				return
			}

			for {
				resp, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					result.err = err
					break
				}

				result.chunkCount++

				if resp.GetReqId() != reqID {
					result.err = io.ErrUnexpectedEOF
					break
				}

				if resp.GetResult() != nil {
					taskID := resp.GetResult().GetTaskId()
					points := resp.GetResult().GetEphemPointsCount()
					result.taskPoints[taskID] += points
					result.totalPoints += points
				}
			}

			result.endTime = time.Now()
			results <- result
		}(int64(i+1), satellites[i])
	}

	// Collect results
	var allResults []ephemResult
	for i := 0; i < numRequests; i++ {
		allResults = append(allResults, <-results)
	}

	// Analyze results
	var errors []error
	for _, r := range allResults {
		if r.err != nil {
			errors = append(errors, r.err)
			t.Errorf("Request %d failed: %v", r.reqID, r.err)
		} else {
			t.Logf("Request %d: %d points, %d chunks, duration=%v",
				r.reqID, r.totalPoints, r.chunkCount, r.endTime.Sub(r.startTime))
		}
	}

	if len(errors) > 0 {
		t.Errorf("Concurrent Ephem with different satellites had %d errors", len(errors))
	} else {
		t.Logf("All %d concurrent Ephem requests with different satellites succeeded", numRequests)
	}
}

func TestAPI_ConcurrentEphemRequests_MultipleTasks(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	const numRequests = 3
	results := make(chan ephemResult, numRequests)

	// Launch concurrent Ephem requests, each with multiple tasks
	for i := 0; i < numRequests; i++ {
		go func(reqID int64) {
			result := ephemResult{
				reqID:      reqID,
				taskPoints: make(map[int64]int64),
				startTime:  time.Now(),
			}

			req := &apiv1.EphemRequest{
				ReqId:     reqID,
				EphemType: apiv1.EphemType_EphemJ2K,
				CommonTimeGrid: &apiv1.EphemTimeGrid{
					TimeStartDs50: 27744.0,
					TimeEndDs50:   27744.02,
					TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepDs50{
						KnownTimeStepDs50: 0.005,
					},
				},
				Tasks: []*apiv1.EphemTask{
					{
						TaskId: reqID*1000 + 1,
						Sat: &apiv1.Satellite{
							NoradId: testNoradID,
							Name:    testSatName,
							TleLn1:  testTLELine1,
							TleLn2:  testTLELine2,
						},
					},
					{
						TaskId: reqID*1000 + 2,
						Sat: &apiv1.Satellite{
							NoradId: testNoradID_2,
							Name:    testSatName_2,
							TleLn1:  testTLELine1_2,
							TleLn2:  testTLELine2_2,
						},
					},
				},
			}

			stream, err := client.Ephem(ctx, req)
			if err != nil {
				result.err = err
				result.endTime = time.Now()
				results <- result
				return
			}

			for {
				resp, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					result.err = err
					break
				}

				result.chunkCount++

				if resp.GetReqId() != reqID {
					result.err = io.ErrUnexpectedEOF
					break
				}

				if resp.GetResult() != nil {
					taskID := resp.GetResult().GetTaskId()
					points := resp.GetResult().GetEphemPointsCount()
					result.taskPoints[taskID] += points
					result.totalPoints += points

					// Verify task ID belongs to this request
					expectedBase := reqID * 1000
					if taskID < expectedBase || taskID > expectedBase+2 {
						result.err = io.ErrUnexpectedEOF
						break
					}
				}
			}

			result.endTime = time.Now()
			results <- result
		}(int64(i + 1))
	}

	// Collect results
	var allResults []ephemResult
	for i := 0; i < numRequests; i++ {
		allResults = append(allResults, <-results)
	}

	// Analyze results
	var errors []error
	for _, r := range allResults {
		if r.err != nil {
			errors = append(errors, r.err)
			t.Errorf("Request %d failed: %v", r.reqID, r.err)
		} else {
			t.Logf("Request %d: %d total points across %d tasks, %d chunks",
				r.reqID, r.totalPoints, len(r.taskPoints), r.chunkCount)
			for taskID, points := range r.taskPoints {
				t.Logf("  Task %d: %d points", taskID, points)
			}
		}

		// Each request should have 2 tasks
		if len(r.taskPoints) != 2 && r.err == nil {
			t.Errorf("Request %d: expected 2 tasks, got %d", r.reqID, len(r.taskPoints))
		}
	}

	if len(errors) > 0 {
		t.Errorf("Concurrent Ephem with multiple tasks had %d errors", len(errors))
	} else {
		t.Logf("All %d concurrent Ephem requests with multiple tasks succeeded", numRequests)
	}
}

func TestAPI_ConcurrentEphemRequests_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	const numRequests = 10
	results := make(chan ephemResult, numRequests)

	testStart := time.Now()

	// Launch many concurrent Ephem requests
	for i := 0; i < numRequests; i++ {
		go func(reqID int64) {
			result := ephemResult{
				reqID:      reqID,
				taskPoints: make(map[int64]int64),
				startTime:  time.Now(),
			}

			// Each request has different time range to ensure different results
			startDs50 := 27744.0 + float64(reqID)*0.1
			endDs50 := startDs50 + 0.05

			req := &apiv1.EphemRequest{
				ReqId:     reqID,
				EphemType: apiv1.EphemType_EphemEci,
				CommonTimeGrid: &apiv1.EphemTimeGrid{
					TimeStartDs50: startDs50,
					TimeEndDs50:   endDs50,
					TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepDs50{
						KnownTimeStepDs50: 0.01,
					},
				},
				Tasks: []*apiv1.EphemTask{
					{
						TaskId: reqID,
						Sat: &apiv1.Satellite{
							NoradId: testNoradID,
							Name:    testSatName,
							TleLn1:  testTLELine1,
							TleLn2:  testTLELine2,
						},
					},
				},
			}

			stream, err := client.Ephem(ctx, req)
			if err != nil {
				result.err = err
				result.endTime = time.Now()
				results <- result
				return
			}

			var firstDs50, lastDs50 float64
			for {
				resp, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					result.err = err
					break
				}

				result.chunkCount++

				if resp.GetResult() != nil {
					points := resp.GetResult().GetEphemPointsCount()
					result.totalPoints += points

					// Track time range
					for _, ephem := range resp.GetResult().GetEphemData() {
						ds50 := ephem.GetDs50Time()
						if firstDs50 == 0 || ds50 < firstDs50 {
							firstDs50 = ds50
						}
						if ds50 > lastDs50 {
							lastDs50 = ds50
						}
					}
				}
			}

			// Verify time range matches expected
			if firstDs50 < startDs50-0.001 || lastDs50 > endDs50+0.001 {
				t.Logf("Request %d: time range mismatch - expected [%f, %f], got [%f, %f]",
					reqID, startDs50, endDs50, firstDs50, lastDs50)
			}

			result.endTime = time.Now()
			results <- result
		}(int64(i + 1))
	}

	// Collect results
	var allResults []ephemResult
	for i := 0; i < numRequests; i++ {
		allResults = append(allResults, <-results)
	}

	totalTestDuration := time.Since(testStart)

	// Analyze results
	var errors []error
	var totalPoints int64
	var totalChunks int
	for _, r := range allResults {
		if r.err != nil {
			errors = append(errors, r.err)
		} else {
			totalPoints += r.totalPoints
			totalChunks += r.chunkCount
		}
	}

	t.Logf("Stress test summary:")
	t.Logf("  Concurrent requests: %d", numRequests)
	t.Logf("  Total test duration: %v", totalTestDuration)
	t.Logf("  Total points generated: %d", totalPoints)
	t.Logf("  Total chunks: %d", totalChunks)
	t.Logf("  Errors: %d", len(errors))

	if len(errors) > 0 {
		t.Errorf("Stress test had %d errors", len(errors))
		for i, err := range errors {
			t.Errorf("  Error %d: %v", i+1, err)
		}
	} else {
		t.Logf("✅ All %d concurrent stress test requests succeeded", numRequests)
	}
}

func TestAPI_ConcurrentMixedPropAndEphem(t *testing.T) {
	ts := newTestServer(t)
	defer ts.close()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	conn, err := ts.dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := apiv1.NewPropagatorClient(conn)

	const numPropRequests = 5
	const numEphemRequests = 3
	totalRequests := numPropRequests + numEphemRequests

	type mixedResult struct {
		reqType string
		reqID   int64
		success bool
		err     error
	}

	results := make(chan mixedResult, totalRequests)

	// Launch concurrent Prop requests
	for i := 0; i < numPropRequests; i++ {
		go func(reqID int64) {
			req := &apiv1.PropRequest{
				ReqId:    reqID,
				TimeType: apiv1.TimeType_TimeDs50,
				Task: &apiv1.PropTask{
					Time: 27744.5 + float64(reqID)*0.01,
					Sat: &apiv1.Satellite{
						NoradId: testNoradID,
						Name:    testSatName,
						TleLn1:  testTLELine1,
						TleLn2:  testTLELine2,
					},
				},
			}

			resp, err := client.Prop(ctx, req)
			if err != nil {
				results <- mixedResult{"Prop", reqID, false, err}
				return
			}

			if resp.GetReqId() != reqID {
				results <- mixedResult{"Prop", reqID, false, io.ErrUnexpectedEOF}
				return
			}

			results <- mixedResult{"Prop", reqID, true, nil}
		}(int64(i + 1))
	}

	// Launch concurrent Ephem requests
	for i := 0; i < numEphemRequests; i++ {
		go func(reqID int64) {
			req := &apiv1.EphemRequest{
				ReqId:     reqID,
				EphemType: apiv1.EphemType_EphemEci,
				CommonTimeGrid: &apiv1.EphemTimeGrid{
					TimeStartDs50: 27744.0,
					TimeEndDs50:   27744.02,
					TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepDs50{
						KnownTimeStepDs50: 0.005,
					},
				},
				Tasks: []*apiv1.EphemTask{
					{
						TaskId: reqID * 10,
						Sat: &apiv1.Satellite{
							NoradId: testNoradID,
							Name:    testSatName,
							TleLn1:  testTLELine1,
							TleLn2:  testTLELine2,
						},
					},
				},
			}

			stream, err := client.Ephem(ctx, req)
			if err != nil {
				results <- mixedResult{"Ephem", reqID, false, err}
				return
			}

			for {
				resp, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					results <- mixedResult{"Ephem", reqID, false, err}
					return
				}

				if resp.GetReqId() != reqID {
					results <- mixedResult{"Ephem", reqID, false, io.ErrUnexpectedEOF}
					return
				}
			}

			results <- mixedResult{"Ephem", reqID, true, nil}
		}(int64(100 + i + 1)) // Different ID range for Ephem
	}

	// Collect results
	var propSuccess, ephemSuccess int
	var propFail, ephemFail int

	for i := 0; i < totalRequests; i++ {
		r := <-results
		if r.success {
			if r.reqType == "Prop" {
				propSuccess++
			} else {
				ephemSuccess++
			}
		} else {
			if r.reqType == "Prop" {
				propFail++
			} else {
				ephemFail++
			}
			t.Errorf("%s request %d failed: %v", r.reqType, r.reqID, r.err)
		}
	}

	t.Logf("Mixed concurrent test results:")
	t.Logf("  Prop: %d/%d succeeded", propSuccess, numPropRequests)
	t.Logf("  Ephem: %d/%d succeeded", ephemSuccess, numEphemRequests)

	if propFail > 0 || ephemFail > 0 {
		t.Errorf("Mixed concurrent test had failures")
	} else {
		t.Logf("✅ All mixed concurrent requests succeeded")
	}
}
