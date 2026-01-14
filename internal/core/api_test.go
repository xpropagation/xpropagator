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
	"testing"
	"time"

	apiv1 "github.com/xpropagation/xpropagator/api/v1/gen"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// =============================================================================
// Prop Request Validation Tests
// =============================================================================

func TestValidatePropRequest_Valid_DS50Time(t *testing.T) {
	req := &apiv1.PropRequest{
		ReqId:    1,
		TimeType: apiv1.TimeType_TimeDs50,
		Task: &apiv1.PropTask{
			Time: 27744.5,
			Sat: &apiv1.Satellite{
				NoradId: 25544,
				Name:    "ISS",
				TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
				TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
			},
		},
	}

	err := validatePropRequest(req)
	if err != nil {
		t.Errorf("Expected valid request, got error: %v", err)
	}
}

func TestValidatePropRequest_Valid_MSETime(t *testing.T) {
	req := &apiv1.PropRequest{
		ReqId:    1,
		TimeType: apiv1.TimeType_TimeMse,
		Task: &apiv1.PropTask{
			Time: 1000.0,
			Sat: &apiv1.Satellite{
				NoradId: 25544,
				Name:    "ISS",
				TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
				TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
			},
		},
	}

	err := validatePropRequest(req)
	if err != nil {
		t.Errorf("Expected valid request, got error: %v", err)
	}
}

func TestValidatePropRequest_Valid_UTCTime(t *testing.T) {
	req := &apiv1.PropRequest{
		ReqId:    1,
		TimeType: apiv1.TimeType_TimeDs50, // Will be converted
		Task: &apiv1.PropTask{
			TimeUtc: timestamppb.New(time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC)),
			Sat: &apiv1.Satellite{
				NoradId: 25544,
				Name:    "ISS",
				TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
				TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
			},
		},
	}

	err := validatePropRequest(req)
	if err != nil {
		t.Errorf("Expected valid request, got error: %v", err)
	}
}

func TestValidatePropRequest_NilTask(t *testing.T) {
	req := &apiv1.PropRequest{
		ReqId:    1,
		TimeType: apiv1.TimeType_TimeDs50,
		Task:     nil,
	}

	err := validatePropRequest(req)
	if err == nil {
		t.Error("Expected error for nil task")
	}
}

func TestValidatePropRequest_NilSatellite(t *testing.T) {
	req := &apiv1.PropRequest{
		ReqId:    1,
		TimeType: apiv1.TimeType_TimeDs50,
		Task: &apiv1.PropTask{
			Time: 27744.5,
			Sat:  nil,
		},
	}

	err := validatePropRequest(req)
	if err == nil {
		t.Error("Expected error for nil satellite")
	}
}

func TestValidatePropRequest_MissingTLELine1(t *testing.T) {
	req := &apiv1.PropRequest{
		ReqId:    1,
		TimeType: apiv1.TimeType_TimeDs50,
		Task: &apiv1.PropTask{
			Time: 27744.5,
			Sat: &apiv1.Satellite{
				NoradId: 25544,
				Name:    "ISS",
				TleLn1:  "",
				TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
			},
		},
	}

	err := validatePropRequest(req)
	if err == nil {
		t.Error("Expected error for missing TLE line 1")
	}
}

func TestValidatePropRequest_MissingTLELine2(t *testing.T) {
	req := &apiv1.PropRequest{
		ReqId:    1,
		TimeType: apiv1.TimeType_TimeDs50,
		Task: &apiv1.PropTask{
			Time: 27744.5,
			Sat: &apiv1.Satellite{
				NoradId: 25544,
				Name:    "ISS",
				TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
				TleLn2:  "",
			},
		},
	}

	err := validatePropRequest(req)
	if err == nil {
		t.Error("Expected error for missing TLE line 2")
	}
}

func TestValidatePropRequest_BothUTCAndDS50Time(t *testing.T) {
	req := &apiv1.PropRequest{
		ReqId:    1,
		TimeType: apiv1.TimeType_TimeDs50,
		Task: &apiv1.PropTask{
			Time:    27744.5, // DS50 time
			TimeUtc: timestamppb.New(time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC)),
			Sat: &apiv1.Satellite{
				NoradId: 25544,
				Name:    "ISS",
				TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
				TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
			},
		},
	}

	err := validatePropRequest(req)
	if err == nil {
		t.Error("Expected error when both UTC and DS50 time are specified")
	}
}

// =============================================================================
// Ephem Request Validation Tests
// =============================================================================

func TestValidateAnalytEphemRequest_Valid_CommonTimeGrid(t *testing.T) {
	req := &apiv1.EphemRequest{
		ReqId:     1,
		EphemType: apiv1.EphemType_EphemEci,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartUtc: timestamppb.New(time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC)),
			TimeEndUtc:   timestamppb.New(time.Date(2025, 12, 28, 0, 0, 0, 0, time.UTC)),
			TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepPeriod{
				KnownTimeStepPeriod: "PT8.5M",
			},
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				Sat: &apiv1.Satellite{
					NoradId: 25544,
					Name:    "ISS",
					TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
					TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
				},
			},
		},
	}

	err := validateAnalytEphemRequest(req)
	if err != nil {
		t.Errorf("Expected valid request, got error: %v", err)
	}
}

func TestValidateAnalytEphemRequest_Valid_J2KFrame(t *testing.T) {
	req := &apiv1.EphemRequest{
		ReqId:     1,
		EphemType: apiv1.EphemType_EphemJ2K,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartDs50: 27744.0,
			TimeEndDs50:   27754.0,
			TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepDs50{
				KnownTimeStepDs50: 0.01,
			},
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				Sat: &apiv1.Satellite{
					NoradId: 25544,
					Name:    "ISS",
					TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
					TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
				},
			},
		},
	}

	err := validateAnalytEphemRequest(req)
	if err != nil {
		t.Errorf("Expected valid request, got error: %v", err)
	}
}

func TestValidateAnalytEphemRequest_NoTasks(t *testing.T) {
	req := &apiv1.EphemRequest{
		ReqId:     1,
		EphemType: apiv1.EphemType_EphemEci,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartDs50: 27744.0,
			TimeEndDs50:   27754.0,
		},
		Tasks: []*apiv1.EphemTask{},
	}

	err := validateAnalytEphemRequest(req)
	if err == nil {
		t.Error("Expected error for no tasks")
	}
}

func TestValidateAnalytEphemRequest_InvalidEphemType(t *testing.T) {
	req := &apiv1.EphemRequest{
		ReqId:     1,
		EphemType: apiv1.EphemType_EphemPlaceholder, // Invalid
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartDs50: 27744.0,
			TimeEndDs50:   27754.0,
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				Sat: &apiv1.Satellite{
					NoradId: 25544,
					Name:    "ISS",
					TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
					TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
				},
			},
		},
	}

	err := validateAnalytEphemRequest(req)
	if err == nil {
		t.Error("Expected error for invalid ephem type")
	}
}

func TestValidateAnalytEphemRequest_NoTimeGrid(t *testing.T) {
	req := &apiv1.EphemRequest{
		ReqId:          1,
		EphemType:      apiv1.EphemType_EphemEci,
		CommonTimeGrid: nil,
		Tasks: []*apiv1.EphemTask{
			{
				TaskId:   1,
				TimeGrid: nil, // No individual time grid
				Sat: &apiv1.Satellite{
					NoradId: 25544,
					Name:    "ISS",
					TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
					TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
				},
			},
		},
	}

	err := validateAnalytEphemRequest(req)
	if err == nil {
		t.Error("Expected error when task has no time grid and no common time grid")
	}
}

func TestValidateAnalytEphemRequest_ConflictingStartTime(t *testing.T) {
	req := &apiv1.EphemRequest{
		ReqId:     1,
		EphemType: apiv1.EphemType_EphemEci,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartUtc:  timestamppb.New(time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC)),
			TimeStartDs50: 27744.0, // Both specified - conflict
			TimeEndUtc:    timestamppb.New(time.Date(2025, 12, 28, 0, 0, 0, 0, time.UTC)),
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				Sat: &apiv1.Satellite{
					NoradId: 25544,
					Name:    "ISS",
					TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
					TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
				},
			},
		},
	}

	err := validateAnalytEphemRequest(req)
	if err == nil {
		t.Error("Expected error for conflicting start time formats")
	}
}

func TestValidateAnalytEphemRequest_ConflictingEndTime(t *testing.T) {
	req := &apiv1.EphemRequest{
		ReqId:     1,
		EphemType: apiv1.EphemType_EphemEci,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartUtc: timestamppb.New(time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC)),
			TimeEndUtc:   timestamppb.New(time.Date(2025, 12, 28, 0, 0, 0, 0, time.UTC)),
			TimeEndDs50:  27754.0, // Both specified - conflict
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				Sat: &apiv1.Satellite{
					NoradId: 25544,
					Name:    "ISS",
					TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
					TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
				},
			},
		},
	}

	err := validateAnalytEphemRequest(req)
	if err == nil {
		t.Error("Expected error for conflicting end time formats")
	}
}

func TestValidateAnalytEphemRequest_ConflictingTimeStep(t *testing.T) {
	req := &apiv1.EphemRequest{
		ReqId:     1,
		EphemType: apiv1.EphemType_EphemEci,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartUtc: timestamppb.New(time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC)),
			TimeEndUtc:   timestamppb.New(time.Date(2025, 12, 28, 0, 0, 0, 0, time.UTC)),
			TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepPeriod{
				KnownTimeStepPeriod: "PT8.5M",
			},
			// KnownTimeStepDs50 would need to be set via reflection since it's a oneof
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				Sat: &apiv1.Satellite{
					NoradId: 25544,
					Name:    "ISS",
					TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
					TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
				},
			},
		},
	}

	// This should pass since only one time step type is set
	err := validateAnalytEphemRequest(req)
	if err != nil {
		t.Errorf("Expected valid request, got error: %v", err)
	}
}

func TestValidateAnalytEphemRequest_MultipleTasks(t *testing.T) {
	req := &apiv1.EphemRequest{
		ReqId:     1,
		EphemType: apiv1.EphemType_EphemEci,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartDs50: 27744.0,
			TimeEndDs50:   27754.0,
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				Sat: &apiv1.Satellite{
					NoradId: 25544,
					Name:    "ISS",
					TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
					TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
				},
			},
			{
				TaskId: 2,
				Sat: &apiv1.Satellite{
					NoradId: 65271,
					Name:    "X-37B",
					TleLn1:  "1 65271U 25183A   25282.36302114 0.00010000  00000-0  55866-4 0    07",
					TleLn2:  "2 65271  48.7951   8.5514 0002000  85.4867 277.3551 15.78566782    05",
				},
			},
		},
	}

	err := validateAnalytEphemRequest(req)
	if err != nil {
		t.Errorf("Expected valid request with multiple tasks, got error: %v", err)
	}
}

func TestValidateAnalytEphemRequest_TaskWithIndividualTimeGrid(t *testing.T) {
	req := &apiv1.EphemRequest{
		ReqId:          1,
		EphemType:      apiv1.EphemType_EphemEci,
		CommonTimeGrid: nil,
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				TimeGrid: &apiv1.EphemTimeGrid{
					TimeStartDs50: 27744.0,
					TimeEndDs50:   27754.0,
				},
				Sat: &apiv1.Satellite{
					NoradId: 25544,
					Name:    "ISS",
					TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
					TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
				},
			},
		},
	}

	err := validateAnalytEphemRequest(req)
	if err != nil {
		t.Errorf("Expected valid request with individual time grid, got error: %v", err)
	}
}

func TestValidateAnalytEphemRequest_TaskWithInvalidSatellite(t *testing.T) {
	req := &apiv1.EphemRequest{
		ReqId:     1,
		EphemType: apiv1.EphemType_EphemEci,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartDs50: 27744.0,
			TimeEndDs50:   27754.0,
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				Sat:    nil, // Invalid satellite
			},
		},
	}

	err := validateAnalytEphemRequest(req)
	if err == nil {
		t.Error("Expected error for nil satellite in task")
	}
}

// =============================================================================
// Satellite Validation Tests
// =============================================================================

func TestValidateSatellite_Valid(t *testing.T) {
	sat := &apiv1.Satellite{
		NoradId: 25544,
		Name:    "ISS",
		TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
		TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
	}

	err := validateSatellite(sat)
	if err != nil {
		t.Errorf("Expected valid satellite, got error: %v", err)
	}
}

func TestValidateSatellite_Nil(t *testing.T) {
	err := validateSatellite(nil)
	if err == nil {
		t.Error("Expected error for nil satellite")
	}
}

func TestValidateSatellite_EmptyTLELine1(t *testing.T) {
	sat := &apiv1.Satellite{
		NoradId: 25544,
		Name:    "ISS",
		TleLn1:  "",
		TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
	}

	err := validateSatellite(sat)
	if err == nil {
		t.Error("Expected error for empty TLE line 1")
	}
}

func TestValidateSatellite_EmptyTLELine2(t *testing.T) {
	sat := &apiv1.Satellite{
		NoradId: 25544,
		Name:    "ISS",
		TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
		TleLn2:  "",
	}

	err := validateSatellite(sat)
	if err == nil {
		t.Error("Expected error for empty TLE line 2")
	}
}

func TestValidateSatellite_OptionalNameEmpty(t *testing.T) {
	sat := &apiv1.Satellite{
		NoradId: 25544,
		Name:    "", // Name is optional
		TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
		TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
	}

	err := validateSatellite(sat)
	if err != nil {
		t.Errorf("Expected valid satellite (name optional), got error: %v", err)
	}
}

// =============================================================================
// Time Conversion Tests
// =============================================================================

func TestUtcToDS50_Epoch(t *testing.T) {
	// DS50 epoch is January 1, 1950 at 12:00:00 UTC
	epoch := time.Date(1950, 1, 1, 12, 0, 0, 0, time.UTC)
	ds50 := UtcToDS50(epoch)

	if ds50 != 0.0 {
		t.Errorf("DS50 at epoch should be 0, got %f", ds50)
	}
}

func TestUtcToDS50_OneDay(t *testing.T) {
	// One day after epoch
	oneDay := time.Date(1950, 1, 2, 12, 0, 0, 0, time.UTC)
	ds50 := UtcToDS50(oneDay)

	if ds50 != 1.0 {
		t.Errorf("DS50 one day after epoch should be 1, got %f", ds50)
	}
}

func TestUtcToDS50_HalfDay(t *testing.T) {
	// Half day after epoch (midnight Jan 2)
	halfDay := time.Date(1950, 1, 2, 0, 0, 0, 0, time.UTC)
	ds50 := UtcToDS50(halfDay)

	if ds50 != 0.5 {
		t.Errorf("DS50 half day after epoch should be 0.5, got %f", ds50)
	}
}

func TestUtcToDS50_ModernDate(t *testing.T) {
	// Test a modern date
	date := time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC)
	ds50 := UtcToDS50(date)

	// Should be around 27744 days since 1950
	if ds50 < 27700 || ds50 > 27800 {
		t.Errorf("DS50 for 2025-12-18 should be around 27744, got %f", ds50)
	}
}

func TestDS50ToUtc_Epoch(t *testing.T) {
	utc := DS50ToUtc(0.0)
	expected := time.Date(1950, 1, 1, 12, 0, 0, 0, time.UTC)

	if !utc.Equal(expected) {
		t.Errorf("DS50=0 should be %v, got %v", expected, utc)
	}
}

func TestDS50ToUtc_OneDay(t *testing.T) {
	utc := DS50ToUtc(1.0)
	expected := time.Date(1950, 1, 2, 12, 0, 0, 0, time.UTC)

	if !utc.Equal(expected) {
		t.Errorf("DS50=1 should be %v, got %v", expected, utc)
	}
}

func TestUtcToDS50_DS50ToUtc_RoundTrip(t *testing.T) {
	original := time.Date(2025, 6, 15, 14, 30, 45, 0, time.UTC)
	ds50 := UtcToDS50(original)
	result := DS50ToUtc(ds50)

	// Allow for small floating point errors
	diff := original.Sub(result)
	if diff < -time.Millisecond || diff > time.Millisecond {
		t.Errorf("Round trip conversion failed: original=%v, result=%v, diff=%v", original, result, diff)
	}
}

// =============================================================================
// Ephemeris Data Conversion Tests
// =============================================================================

func TestFlatEphemsToEphemDataArr_Valid(t *testing.T) {
	// 2 ephemeris points with 7 values each
	flat := []float64{
		27744.0, 1000.0, 2000.0, 3000.0, 1.0, 2.0, 3.0,
		27744.1, 1001.0, 2001.0, 3001.0, 1.1, 2.1, 3.1,
	}

	result, err := flatEphemsToEphemDataArr(flat)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("Expected 2 ephemeris points, got %d", len(result))
	}

	// Check first point
	if result[0].Ds50Time != 27744.0 {
		t.Errorf("First point DS50Time = %f, want 27744.0", result[0].Ds50Time)
	}
	if result[0].X != 1000.0 {
		t.Errorf("First point X = %f, want 1000.0", result[0].X)
	}
	if result[0].Y != 2000.0 {
		t.Errorf("First point Y = %f, want 2000.0", result[0].Y)
	}
	if result[0].Z != 3000.0 {
		t.Errorf("First point Z = %f, want 3000.0", result[0].Z)
	}
	if result[0].Vx != 1.0 {
		t.Errorf("First point Vx = %f, want 1.0", result[0].Vx)
	}
	if result[0].Vy != 2.0 {
		t.Errorf("First point Vy = %f, want 2.0", result[0].Vy)
	}
	if result[0].Vz != 3.0 {
		t.Errorf("First point Vz = %f, want 3.0", result[0].Vz)
	}

	// Check second point
	if result[1].Ds50Time != 27744.1 {
		t.Errorf("Second point DS50Time = %f, want 27744.1", result[1].Ds50Time)
	}
}

func TestFlatEphemsToEphemDataArr_TooShort(t *testing.T) {
	flat := []float64{1.0, 2.0, 3.0} // Less than 7 elements

	_, err := flatEphemsToEphemDataArr(flat)
	if err == nil {
		t.Error("Expected error for array shorter than 7 elements")
	}
}

func TestFlatEphemsToEphemDataArr_NotMultipleOf7(t *testing.T) {
	flat := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0} // 8 elements, not multiple of 7

	_, err := flatEphemsToEphemDataArr(flat)
	if err == nil {
		t.Error("Expected error for array length not multiple of 7")
	}
}

func TestFlatEphemsToEphemDataArr_SinglePoint(t *testing.T) {
	flat := []float64{27744.5, 6632.4, -963.4, -301.9, 0.98, 4.99, 5.79}

	result, err := flatEphemsToEphemDataArr(flat)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 ephemeris point, got %d", len(result))
	}
}

// =============================================================================
// Time Grid Resolution Tests
// =============================================================================

func TestResolveTimeGrid_UsesTaskGrid(t *testing.T) {
	taskGrid := &apiv1.EphemTimeGrid{
		TimeStartDs50: 100.0,
		TimeEndDs50:   200.0,
	}
	commonGrid := &apiv1.EphemTimeGrid{
		TimeStartDs50: 0.0,
		TimeEndDs50:   50.0,
	}

	task := &apiv1.EphemTask{
		TimeGrid: taskGrid,
	}
	req := &apiv1.EphemRequest{
		CommonTimeGrid: commonGrid,
	}

	result := resolveTimeGrid(task, req)

	if result != taskGrid {
		t.Error("Should use task's time grid when specified")
	}
}

func TestResolveTimeGrid_UsesCommonGrid(t *testing.T) {
	commonGrid := &apiv1.EphemTimeGrid{
		TimeStartDs50: 0.0,
		TimeEndDs50:   50.0,
	}

	task := &apiv1.EphemTask{
		TimeGrid: nil,
	}
	req := &apiv1.EphemRequest{
		CommonTimeGrid: commonGrid,
	}

	result := resolveTimeGrid(task, req)

	if result != commonGrid {
		t.Error("Should use common time grid when task grid is nil")
	}
}

// =============================================================================
// Time Step Tests
// =============================================================================

func TestIsDynamicTimeStep_True(t *testing.T) {
	grid := &apiv1.EphemTimeGrid{
		TimeStepType: &apiv1.EphemTimeGrid_DynamicTimeStep{
			DynamicTimeStep: true,
		},
	}

	if !isDynamicTimeStep(grid) {
		t.Error("Expected dynamic time step to be true")
	}
}

func TestIsDynamicTimeStep_False(t *testing.T) {
	grid := &apiv1.EphemTimeGrid{
		TimeStepType: &apiv1.EphemTimeGrid_DynamicTimeStep{
			DynamicTimeStep: false,
		},
	}

	if isDynamicTimeStep(grid) {
		t.Error("Expected dynamic time step to be false")
	}
}

func TestIsDynamicTimeStep_OtherType(t *testing.T) {
	grid := &apiv1.EphemTimeGrid{
		TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepPeriod{
			KnownTimeStepPeriod: "PT1M",
		},
	}

	if isDynamicTimeStep(grid) {
		t.Error("Expected dynamic time step to be false for period type")
	}
}

func TestGetKnownTimeStep_Period(t *testing.T) {
	grid := &apiv1.EphemTimeGrid{
		TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepPeriod{
			KnownTimeStepPeriod: "PT10M", // 10 minutes
		},
	}

	step := getKnownTimeStep(grid)

	if step != 10.0 {
		t.Errorf("Expected 10 minutes, got %f", step)
	}
}

func TestGetKnownTimeStep_DS50(t *testing.T) {
	grid := &apiv1.EphemTimeGrid{
		TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepDs50{
			KnownTimeStepDs50: 0.01, // 0.01 days = 14.4 minutes
		},
	}

	step := getKnownTimeStep(grid)

	expected := 0.01 * 1440 // Convert to minutes
	if step != expected {
		t.Errorf("Expected %f minutes, got %f", expected, step)
	}
}

func TestGetKnownTimeStep_Dynamic_ReturnsZero(t *testing.T) {
	grid := &apiv1.EphemTimeGrid{
		TimeStepType: &apiv1.EphemTimeGrid_DynamicTimeStep{
			DynamicTimeStep: true,
		},
	}

	step := getKnownTimeStep(grid)

	if step != 0 {
		t.Errorf("Expected 0 for dynamic time step, got %f", step)
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkValidatePropRequest(b *testing.B) {
	req := &apiv1.PropRequest{
		ReqId:    1,
		TimeType: apiv1.TimeType_TimeDs50,
		Task: &apiv1.PropTask{
			Time: 27744.5,
			Sat: &apiv1.Satellite{
				NoradId: 25544,
				Name:    "ISS",
				TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
				TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validatePropRequest(req)
	}
}

func BenchmarkValidateEphemRequest(b *testing.B) {
	req := &apiv1.EphemRequest{
		ReqId:     1,
		EphemType: apiv1.EphemType_EphemEci,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartDs50: 27744.0,
			TimeEndDs50:   27754.0,
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 1,
				Sat: &apiv1.Satellite{
					NoradId: 25544,
					Name:    "ISS",
					TleLn1:  "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
					TleLn2:  "2 25544  51.6442 208.5453 0003439  47.4501  63.9527 15.48881544315506",
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validateAnalytEphemRequest(req)
	}
}

func BenchmarkUtcToDS50(b *testing.B) {
	t := time.Date(2025, 12, 18, 14, 30, 45, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = UtcToDS50(t)
	}
}

func BenchmarkFlatEphemsToEphemDataArr(b *testing.B) {
	// 100 ephemeris points
	flat := make([]float64, 700)
	for i := 0; i < 700; i++ {
		flat[i] = float64(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = flatEphemsToEphemDataArr(flat)
	}
}
