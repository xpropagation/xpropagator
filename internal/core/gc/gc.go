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

package gc

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xpropagation/xpropagator/internal/config"
	"github.com/xpropagation/xpropagator/internal/core_helpers"
	"github.com/xpropagation/xpropagator/internal/dllcore"
	"github.com/xpropagation/xpropagator/internal/values"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module("gc",
	fx.Provide(NewGCFromConfig),
)

var catalogMu sync.Mutex

type SatEntry struct {
	key      int64
	satNum   int32
	lastUsed time.Time
	refs     int
}

type GC struct {
	mu            sync.Mutex
	loaded        map[int64]*SatEntry // key -> entry
	maxLoaded     int
	idleTTL       time.Duration
	sweepInterval time.Duration

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewGC(maxLoaded int, idleTTL, sweepInterval time.Duration) *GC {
	if maxLoaded < 1 {
		maxLoaded = values.DefaultMaxLoadedSatsGc
	}
	if idleTTL <= 0 {
		idleTTL, _ = time.ParseDuration(values.DefaultIdleTTLGcMin)
	}
	if sweepInterval <= 0 {
		sweepInterval, _ = time.ParseDuration(values.DefaultSweepIntervalGcMin)
	}
	gc := &GC{
		loaded:        make(map[int64]*SatEntry, maxLoaded),
		maxLoaded:     maxLoaded,
		idleTTL:       idleTTL,
		sweepInterval: sweepInterval,
		stopCh:        make(chan struct{}),
	}
	gc.wg.Add(1)
	go gc.sweeper()
	return gc
}

func NewGCFromConfig(cfg *config.Config, logger *zap.Logger, lc fx.Lifecycle) *GC {
	gc := NewGC(
		cfg.GC.MaxLoadedSatsGc,
		cfg.GC.IdleTTLGcMin,
		cfg.GC.SweepIntervalGcMin,
	)

	logger.Info("satellite GC initialized",
		zap.Int("max_loaded_sats", cfg.GC.MaxLoadedSatsGc),
		zap.Duration("idle_ttl", cfg.GC.IdleTTLGcMin),
		zap.Duration("sweep_interval", cfg.GC.SweepIntervalGcMin),
	)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("stopping satellite GC")
			gc.Close()
			logger.Info("satellite GC stopped")
			return nil
		},
	})

	return gc
}

func (g *GC) Close() {
	close(g.stopCh)
	g.wg.Wait()
}

func (g *GC) addTLE(ctx context.Context, ln1, ln2 string) (int64, error) {
	satNum, err := parseSatNum(ln1)
	if err != nil {
		return 0, fmt.Errorf("failed to parse satellite number from TLE first line: %w", err)
	}

	var existing int64
	if err = core_helpers.WithDllCall(ctx, func() int {
		existing = dllcore.TleGetSatKey(int32(satNum))
		return 0
	}); err != nil {
		return 0, err
	}
	if existing != -1 {
		return existing, nil
	}

	var key int64
	if err = core_helpers.WithDllCall(ctx, func() int {
		key = dllcore.TleAddSatFrLines(ln1, ln2)
		if key <= 0 {
			return 1
		}
		return 0
	}); err != nil {
		return 0, err
	}
	return key, nil
}

func (g *GC) addOrInitSat(ctx context.Context, l1, l2 string) (int64, error) {
	catalogMu.Lock()
	key, err := g.addTLE(ctx, l1, l2)
	if err != nil {
		catalogMu.Unlock()
		return 0, err
	}
	mu := core_helpers.RwFor(key)
	mu.Lock()
	catalogMu.Unlock()
	defer mu.Unlock()

	if err := core_helpers.WithDllCall(ctx, func() int { return dllcore.Sgp4InitSat(key) }); err != nil {
		return 0, err
	}
	return key, nil
}

func (g *GC) Acquire(ctx context.Context, l1, l2 string) (int64, func(), error) {
	satNum, err := parseSatNum(l1)
	if err != nil {
		return 0, nil, err
	}

	catalogMu.Lock()
	var existing int64 = -1
	if err := core_helpers.WithDllCall(ctx, func() int {
		existing = dllcore.TleGetSatKey(satNum)
		return 0
	}); err != nil {
		catalogMu.Unlock()
		return 0, nil, err
	}

	if existing == -1 {
		catalogMu.Unlock()

		victims := g.evictLRUIfNeeded(1)
		g.removeVictims(ctx, victims)

		key, err := g.addOrInitSat(ctx, l1, l2)
		if err != nil {
			return 0, nil, err
		}

		mu := core_helpers.RwFor(key)
		mu.RLock()

		g.mu.Lock()
		e := g.loaded[key]
		if e == nil {
			e = &SatEntry{key: key, satNum: satNum, lastUsed: time.Now(), refs: 1}
			g.loaded[key] = e
		} else {
			e.refs++
			e.lastUsed = time.Now()
		}
		g.mu.Unlock()

		return key, g.makeReleaser(key, mu.RUnlock), nil
	}

	key := existing
	mu := core_helpers.RwFor(key)
	mu.RLock()
	catalogMu.Unlock()

	g.mu.Lock()
	e := g.loaded[key]
	if e == nil {
		e = &SatEntry{key: key, satNum: satNum, lastUsed: time.Now(), refs: 1}
		g.loaded[key] = e
	} else {
		e.refs++
		e.lastUsed = time.Now()
	}
	g.mu.Unlock()

	return key, g.makeReleaser(key, mu.RUnlock), nil
}

func (g *GC) makeReleaser(key int64, runlock func()) func() {
	return func() {
		now := time.Now()
		g.mu.Lock()
		if e := g.loaded[key]; e != nil && e.refs > 0 {
			e.refs--
			e.lastUsed = now
		}
		g.mu.Unlock()
		if runlock != nil {
			runlock()
		}
	}
}

func (g *GC) evictLRUIfNeeded(need int) []int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	excess := len(g.loaded) + need - g.maxLoaded
	if excess <= 0 {
		return nil
	}

	type pair struct {
		key int64
		ts  time.Time
	}
	candidates := make([]pair, 0, len(g.loaded))
	for key, e := range g.loaded {
		if e.refs == 0 {
			candidates = append(candidates, pair{key, e.lastUsed})
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ts.Before(candidates[j].ts) })

	if excess > len(candidates) {
		excess = len(candidates)
	}
	victims := make([]int64, 0, excess)
	for i := 0; i < excess; i++ {
		victims = append(victims, candidates[i].key)
	}
	return victims
}

func (g *GC) sweeper() {
	defer g.wg.Done()
	t := time.NewTicker(g.sweepInterval)
	defer t.Stop()

	for {
		select {
		case <-g.stopCh:
			return
		case now := <-t.C:
			var victims []int64
			g.mu.Lock()
			for key, e := range g.loaded {
				if e.refs == 0 && now.Sub(e.lastUsed) > g.idleTTL {
					victims = append(victims, key)
				}
			}
			g.mu.Unlock()
			g.removeVictims(context.Background(), victims)
		}
	}
}

func (g *GC) removeVictims(ctx context.Context, victims []int64) {
	for _, key := range victims {
		mu := core_helpers.RwFor(key)
		mu.Lock()
		catalogMu.Lock()

		shouldRemove := false
		g.mu.Lock()
		if e := g.loaded[key]; e != nil && e.refs == 0 {
			shouldRemove = true
		}
		g.mu.Unlock()

		if shouldRemove {
			_ = core_helpers.WithDllCall(ctx, func() int { return dllcore.Sgp4RemoveSat(key) })
			_ = core_helpers.WithDllCall(ctx, func() int { return dllcore.TleRemoveSat(key) })

			g.mu.Lock()
			delete(g.loaded, key)
			g.mu.Unlock()
		}

		catalogMu.Unlock()
		mu.Unlock()
	}
}

func (g *GC) WaitAllReleased(ctx context.Context) error {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			g.mu.Lock()
			allZero := true
			for _, e := range g.loaded {
				if e.refs > 0 {
					allZero = false
					break
				}
			}
			g.mu.Unlock()
			if allZero {
				return nil
			}
		}
	}
}

func (g *GC) RemoveAll(ctx context.Context) error {
	catalogMu.Lock()
	defer catalogMu.Unlock()

	if err := g.WaitAllReleased(ctx); err != nil {
		return err
	}

	if err := core_helpers.WithDllCall(ctx, func() int { return dllcore.Sgp4RemoveAllSats() }); err != nil {
		return err
	}
	if err := core_helpers.WithDllCall(ctx, func() int { return dllcore.TleRemoveAllSats() }); err != nil {
		return err
	}

	g.mu.Lock()
	g.loaded = make(map[int64]*SatEntry, g.maxLoaded)
	g.mu.Unlock()

	return nil
}

func parseSatNum(l1 string) (int32, error) {
	if len(l1) < 7 {
		return 0, fmt.Errorf("invalid TLE first line length")
	}
	s := strings.TrimSpace(l1[2:7])
	if len(s) != 5 {
		return 0, fmt.Errorf("invalid satnum length: %s", s)
	}

	// Try numeric first (legacy 5-digit)
	if n, err := strconv.Atoi(s); err == nil {
		if n >= 1 && n <= 99999 {
			return int32(n), nil
		}
	}

	// Alpha-5 parsing: Letter (A-Z except I,O) + 4 digits
	if len(s) == 5 {
		letter := s[0]
		digits := s[1:]

		// Validate letter (A-Z excluding I,O per USSF spec)
		if letter >= 'A' && letter <= 'Z' && letter != 'I' && letter != 'O' {
			num, err := strconv.Atoi(digits)
			if err == nil && num >= 0 && num <= 9999 {
				// A=10, B=11, ..., Z=35
				prefix := int(letter-'A'+10) * 10000
				satnum := prefix + num
				if satnum >= 100000 && satnum <= 359999 { // Valid Alpha-5 range
					return int32(satnum), nil
				}
			}
		}
	}

	return 0, fmt.Errorf("parse satNum: invalid format %q", s)
}
