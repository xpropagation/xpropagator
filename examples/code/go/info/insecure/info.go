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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
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

	infoResp, err := c.Info(context.Background(), &emptypb.Empty{})
	if err != nil {
		log.Fatal("failed to request api.v1.Propagator.Info : ", err)
	}

	log.Printf("api.v1.Propagator.Info response:\nName: %s,\nVersion: %s,\nCommit: %s,\nBuildDate: %s,\nAstroStdLibInfo: %s,\nSgp4LibInfo: %s,\nTimestamp: %s",
		infoResp.GetName(), infoResp.GetVersion(), infoResp.GetCommit(), infoResp.GetBuildDate(), infoResp.GetAstroStdLibInfo(), infoResp.GetSgp4LibInfo(), infoResp.GetTimestamp().AsTime().String())
}
