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

package main

import (
	"context"
	apiv1 "github.com/xpropagation/xpropagator/api/v1/gen"
	"github.com/xpropagation/xpropagator/examples/code/go/helper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"log"
	"time"
)

func main() {
	tlsCfg, err := helper.GetTLSConfig()
	if err != nil {
		log.Fatalf("Failed to create TLS config: %s", err)
	}

	creds := credentials.NewTLS(tlsCfg)

	cc, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatal("Failed to create XPropagator gRPC client: ", err)
	}
	defer func() {
		_ = cc.Close()
	}()

	c := apiv1.NewPropagatorClient(cc)

	timeStart := time.Now()

	ephemReq := &apiv1.EphemRequest{
		ReqId:     1,
		EphemType: apiv1.EphemType_EphemJ2K,
		CommonTimeGrid: &apiv1.EphemTimeGrid{
			TimeStartUtc: timestamppb.New(time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC)),
			TimeEndUtc:   timestamppb.New(time.Date(2025, 12, 28, 0, 0, 0, 0, time.UTC)),
			TimeStepType: &apiv1.EphemTimeGrid_DynamicTimeStep{
				DynamicTimeStep: true,
			},
		},
		Tasks: []*apiv1.EphemTask{
			{
				TaskId: 10,
				Sat: &apiv1.Satellite{
					NoradId: 65271,
					Name:    "X-37B Orbital Test Vehicle 8 (OTV 8)",
					TleLn1:  "1 65271U 25183A   25282.36302114 0.00010000  00000-0  55866-4 0    07",
					TleLn2:  "2 65271  48.7951   8.5514 0002000  85.4867 277.3551 15.78566782    05",
				},
			},
			{
				TaskId: 20,
				TimeGrid: &apiv1.EphemTimeGrid{
					TimeStartUtc: timestamppb.New(time.Date(2025, 12, 29, 0, 0, 0, 0, time.UTC)),
					TimeEndUtc:   timestamppb.New(time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC)),
					TimeStepType: &apiv1.EphemTimeGrid_DynamicTimeStep{
						DynamicTimeStep: true,
					},
				},
				Sat: &apiv1.Satellite{
					NoradId: 2000,
					Name:    "Satellite B",
					TleLn1:  "1 49220U 21089B   24290.21456789  .00014567  00000-0  62458-3 0  9991",
					TleLn2:  "2 49220  53.0021 320.8765 0078456  42.6543 317.8845 14.87654321987654",
				},
			},
			{
				TaskId: 30,
				Sat: &apiv1.Satellite{
					NoradId: 3000,
					Name:    "Satellite C",
					TleLn1:  "1 60123U 24150C   24290.84567890  .00000023  00000-0  15987-5 0  9993",
					TleLn2:  "2 60123  28.5123 210.2345 0019876 102.3456 257.6543 12.34567890123456",
				},
			},
		},
	}

	ctx := context.Background()

	stream, err := c.Ephem(ctx, ephemReq)
	if err != nil {
		log.Fatalf("failed to request api.v1.Propagator.Ephem: %s", err)
	}
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("failed to receive stream chunk from api.v1.Propagator.Ephem: %s", err)
		}

		log.Printf("api.v1.Propagator.Ephem stream chunk received:\n"+
			"ReqId: %d\n"+
			"TaskId: %d\n"+
			"StreamId: %d\n"+
			"StreamChunkId: %d\n"+
			"EphemerisCount: %v",
			resp.GetReqId(),
			resp.GetResult().GetTaskId(),
			resp.GetStreamId(),
			resp.GetStreamChunkId(),
			resp.GetResult().EphemPointsCount,
		)

		for _, ephemData := range resp.GetResult().EphemData {
			log.Println(ephemData.String())
		}
	}

	log.Printf("api.v1.Propagator.Ephem done, time took: %s", time.Since(timeStart))
}
