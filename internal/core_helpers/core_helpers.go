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

package core_helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/xpropagation/xpropagator/internal/dllcore"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"sync"
	"sync/atomic"
)

var satLocks sync.Map // satKey -> *sync.RWMutex

func RwFor(key int64) *sync.RWMutex {
	v, _ := satLocks.LoadOrStore(key, &sync.RWMutex{})
	return v.(*sync.RWMutex)
}

func ProtoField(key string, msg proto.Message) zap.Field {
	b, _ := protojson.Marshal(msg)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	return zap.Any(key, m)
}

var (
	errMu sync.Mutex
)

var (
	curGate atomic.Value // *gate
)

type gate struct{ ch chan struct{} }

// вызывать один раз при старте процесса (или перед первым RPC)
func SetDllGate(n int) {
	if n < 1 {
		n = 1
	}
	g := &gate{ch: make(chan struct{}, n)}
	curGate.Store(g)
}

func ensureGate() *gate {
	g, _ := curGate.Load().(*gate)
	if g == nil {
		SetDllGate(1)
		g, _ = curGate.Load().(*gate)
	}
	return g
}

func WithDllCall(ctx context.Context, call func() int) error {
	g := ensureGate()

	select {
	case g.ch <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}

	rc := call()
	<-g.ch

	if rc != 0 {
		errMu.Lock()
		msg := dllcore.GetLastErrMsg()
		errMu.Unlock()
		return fmt.Errorf("rc=%d: %s", rc, msg)
	}
	return nil
}
