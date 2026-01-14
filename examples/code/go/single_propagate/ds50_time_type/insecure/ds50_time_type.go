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
	"log"
	"time"

	apiv1 "github.com/xpropagation/xpropagator/api/v1/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cc, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Failed to create XPropagator gRPC client: ", err)
	}
	defer func() {
		_ = cc.Close()
	}()

	c := apiv1.NewPropagatorClient(cc)

	timeStart := time.Now()

	propReq := &apiv1.PropRequest{
		ReqId:    1,
		TimeType: apiv1.TimeType_TimeDs50,
		Task: &apiv1.PropTask{
			Time: 27744.5,
			Sat: &apiv1.Satellite{
				NoradId: 65271,
				Name:    "X-37B Orbital Test Vehicle 8 (OTV 8)",
				TleLn1:  "1 65271U 25183A   25282.36302114 0.00010000  00000-0  55866-4 0    07",
				TleLn2:  "2 65271  48.7951   8.5514 0002000  85.4867 277.3551 15.78566782    05",
			},
		},
	}

	resp, err := c.Prop(context.Background(), propReq)
	if err != nil {
		log.Fatalf("failed to request api.v1.Propagator.Prop: %s", err)
	}

	log.Printf("api.v1.Propagator.Prop done, time took: %s\n%s", time.Since(timeStart), resp.GetResult())
}
