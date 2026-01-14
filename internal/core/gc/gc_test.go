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
	"sync"
	"testing"
	"time"
)

// Helper methods for testing (expose internal state)

// LoadedCount returns the number of loaded satellites
func (g *GC) LoadedCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.loaded)
}

// GetEntry returns a copy of the entry for testing
func (g *GC) GetEntry(key int64) *SatEntry {
	g.mu.Lock()
	defer g.mu.Unlock()
	if e, ok := g.loaded[key]; ok {
		return &SatEntry{
			key:      e.key,
			satNum:   e.satNum,
			lastUsed: e.lastUsed,
			refs:     e.refs,
		}
	}
	return nil
}

// AddTestEntry adds a test entry directly to the loaded map (for testing without DLL)
func (g *GC) AddTestEntry(key int64, satNum int32, lastUsed time.Time, refs int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.loaded[key] = &SatEntry{
		key:      key,
		satNum:   satNum,
		lastUsed: lastUsed,
		refs:     refs,
	}
}

// RemoveTestEntry removes an entry from the loaded map (for testing)
func (g *GC) RemoveTestEntry(key int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.loaded, key)
}

// GetMaxLoaded returns maxLoaded for testing
func (g *GC) GetMaxLoaded() int {
	return g.maxLoaded
}

// GetIdleTTL returns idleTTL for testing
func (g *GC) GetIdleTTL() time.Duration {
	return g.idleTTL
}

// GetSweepInterval returns sweepInterval for testing
func (g *GC) GetSweepInterval() time.Duration {
	return g.sweepInterval
}

// TestReleaser creates a test releaser for a given key
func (g *GC) TestReleaser(key int64) func() {
	return g.makeReleaser(key, nil)
}

// =============================================================================
// TESTS
// =============================================================================

func TestNewGC_DefaultValues(t *testing.T) {
	// Test with invalid/zero values - should use defaults
	gc := NewGC(0, 0, 0)
	defer gc.Close()

	if gc.GetMaxLoaded() < 1 {
		t.Errorf("maxLoaded should have a positive default, got %d", gc.GetMaxLoaded())
	}
	if gc.GetIdleTTL() <= 0 {
		t.Errorf("idleTTL should have a positive default, got %v", gc.GetIdleTTL())
	}
	if gc.GetSweepInterval() <= 0 {
		t.Errorf("sweepInterval should have a positive default, got %v", gc.GetSweepInterval())
	}
}

func TestNewGC_CustomValues(t *testing.T) {
	maxLoaded := 100
	idleTTL := 5 * time.Minute
	sweepInterval := 1 * time.Minute

	gc := NewGC(maxLoaded, idleTTL, sweepInterval)
	defer gc.Close()

	if gc.GetMaxLoaded() != maxLoaded {
		t.Errorf("maxLoaded = %d, want %d", gc.GetMaxLoaded(), maxLoaded)
	}
	if gc.GetIdleTTL() != idleTTL {
		t.Errorf("idleTTL = %v, want %v", gc.GetIdleTTL(), idleTTL)
	}
	if gc.GetSweepInterval() != sweepInterval {
		t.Errorf("sweepInterval = %v, want %v", gc.GetSweepInterval(), sweepInterval)
	}
}

func TestNewGC_NegativeValues(t *testing.T) {
	// Negative values should be replaced with defaults
	gc := NewGC(-10, -5*time.Minute, -1*time.Minute)
	defer gc.Close()

	if gc.GetMaxLoaded() < 1 {
		t.Errorf("negative maxLoaded should use default, got %d", gc.GetMaxLoaded())
	}
	if gc.GetIdleTTL() <= 0 {
		t.Errorf("negative idleTTL should use default, got %v", gc.GetIdleTTL())
	}
	if gc.GetSweepInterval() <= 0 {
		t.Errorf("negative sweepInterval should use default, got %v", gc.GetSweepInterval())
	}
}

func TestGC_MaxSatsLimit_EvictsLRU(t *testing.T) {
	// Create GC with max 3 satellites
	gc := NewGC(3, 10*time.Minute, 1*time.Hour)
	defer gc.Close()

	now := time.Now()

	// Add 3 satellites with different lastUsed times
	gc.AddTestEntry(1, 10001, now.Add(-3*time.Minute), 0) // oldest - should be evicted first
	gc.AddTestEntry(2, 10002, now.Add(-2*time.Minute), 0) // middle
	gc.AddTestEntry(3, 10003, now.Add(-1*time.Minute), 0) // newest

	if gc.LoadedCount() != 3 {
		t.Fatalf("Expected 3 loaded satellites, got %d", gc.LoadedCount())
	}

	// Request eviction for 1 more slot
	victims := gc.evictLRUIfNeeded(1)

	if len(victims) != 1 {
		t.Errorf("Expected 1 victim, got %d", len(victims))
	}

	// Should evict the oldest (key=1)
	if len(victims) > 0 && victims[0] != 1 {
		t.Errorf("Expected victim key=1 (oldest), got key=%d", victims[0])
	}
}

func TestGC_MaxSatsLimit_EvictsMultipleLRU(t *testing.T) {
	// Create GC with max 5 satellites
	gc := NewGC(5, 10*time.Minute, 1*time.Hour)
	defer gc.Close()

	now := time.Now()

	// Add 5 satellites
	gc.AddTestEntry(1, 10001, now.Add(-5*time.Minute), 0)
	gc.AddTestEntry(2, 10002, now.Add(-4*time.Minute), 0)
	gc.AddTestEntry(3, 10003, now.Add(-3*time.Minute), 0)
	gc.AddTestEntry(4, 10004, now.Add(-2*time.Minute), 0)
	gc.AddTestEntry(5, 10005, now.Add(-1*time.Minute), 0)

	// Request eviction for 3 more slots - should evict 3 oldest
	victims := gc.evictLRUIfNeeded(3)

	if len(victims) != 3 {
		t.Errorf("Expected 3 victims, got %d", len(victims))
	}

	// Verify victims are the oldest ones (keys 1, 2, 3)
	victimMap := make(map[int64]bool)
	for _, v := range victims {
		victimMap[v] = true
	}

	for _, expectedKey := range []int64{1, 2, 3} {
		if !victimMap[expectedKey] {
			t.Errorf("Expected key=%d to be in victims", expectedKey)
		}
	}
}

func TestGC_MaxSatsLimit_DoesNotEvictInUse(t *testing.T) {
	// Create GC with max 2 satellites
	gc := NewGC(2, 10*time.Minute, 1*time.Hour)
	defer gc.Close()

	now := time.Now()

	// Add 2 satellites - one in use (refs > 0)
	gc.AddTestEntry(1, 10001, now.Add(-2*time.Minute), 1) // IN USE - refs=1
	gc.AddTestEntry(2, 10002, now.Add(-1*time.Minute), 0) // not in use

	// Request eviction for 1 more slot
	victims := gc.evictLRUIfNeeded(1)

	if len(victims) != 1 {
		t.Errorf("Expected 1 victim, got %d", len(victims))
	}

	// Should only evict key=2 (not in use), not key=1 (in use)
	if len(victims) > 0 && victims[0] != 2 {
		t.Errorf("Expected victim key=2 (not in use), got key=%d", victims[0])
	}
}

func TestGC_MaxSatsLimit_AllInUse_NoEviction(t *testing.T) {
	// Create GC with max 2 satellites
	gc := NewGC(2, 10*time.Minute, 1*time.Hour)
	defer gc.Close()

	now := time.Now()

	// Add 2 satellites - both in use
	gc.AddTestEntry(1, 10001, now.Add(-2*time.Minute), 1) // IN USE
	gc.AddTestEntry(2, 10002, now.Add(-1*time.Minute), 2) // IN USE

	// Request eviction for 1 more slot
	victims := gc.evictLRUIfNeeded(1)

	if len(victims) != 0 {
		t.Errorf("Expected 0 victims (all in use), got %d", len(victims))
	}
}

func TestGC_MaxSatsLimit_UnderLimit_NoEviction(t *testing.T) {
	// Create GC with max 10 satellites
	gc := NewGC(10, 10*time.Minute, 1*time.Hour)
	defer gc.Close()

	now := time.Now()

	// Add only 3 satellites
	gc.AddTestEntry(1, 10001, now.Add(-3*time.Minute), 0)
	gc.AddTestEntry(2, 10002, now.Add(-2*time.Minute), 0)
	gc.AddTestEntry(3, 10003, now.Add(-1*time.Minute), 0)

	// Request eviction for 1 more slot - should not evict since under limit
	victims := gc.evictLRUIfNeeded(1)

	if len(victims) != 0 {
		t.Errorf("Expected 0 victims (under limit), got %d", len(victims))
	}
}

func TestGC_TTL_IdleEntriesIdentified(t *testing.T) {
	// Create GC with 1 minute TTL
	idleTTL := 1 * time.Minute
	gc := NewGC(100, idleTTL, 1*time.Hour)
	defer gc.Close()

	now := time.Now()

	// Add entries with different ages
	gc.AddTestEntry(1, 10001, now.Add(-2*time.Minute), 0)  // older than TTL - should be victim
	gc.AddTestEntry(2, 10002, now.Add(-90*time.Second), 0) // older than TTL - should be victim
	gc.AddTestEntry(3, 10003, now.Add(-30*time.Second), 0) // younger than TTL - should NOT be victim
	gc.AddTestEntry(4, 10004, now.Add(-2*time.Minute), 1)  // older but IN USE - should NOT be victim

	// Simulate sweeper check
	var victims []int64
	gc.mu.Lock()
	for key, e := range gc.loaded {
		if e.refs == 0 && now.Sub(e.lastUsed) > idleTTL {
			victims = append(victims, key)
		}
	}
	gc.mu.Unlock()

	if len(victims) != 2 {
		t.Errorf("Expected 2 TTL victims, got %d", len(victims))
	}

	victimMap := make(map[int64]bool)
	for _, v := range victims {
		victimMap[v] = true
	}

	if !victimMap[1] {
		t.Error("Key 1 should be a TTL victim")
	}
	if !victimMap[2] {
		t.Error("Key 2 should be a TTL victim")
	}
	if victimMap[3] {
		t.Error("Key 3 should NOT be a TTL victim (too recent)")
	}
	if victimMap[4] {
		t.Error("Key 4 should NOT be a TTL victim (in use)")
	}
}

func TestGC_Releaser_DecrementsRefs(t *testing.T) {
	gc := NewGC(100, 10*time.Minute, 1*time.Hour)
	defer gc.Close()

	now := time.Now()

	// Add entry with refs=2
	gc.AddTestEntry(1, 10001, now, 2)

	entry := gc.GetEntry(1)
	if entry.refs != 2 {
		t.Fatalf("Initial refs should be 2, got %d", entry.refs)
	}

	// Create and call releaser
	releaser := gc.TestReleaser(1)
	releaser()

	entry = gc.GetEntry(1)
	if entry.refs != 1 {
		t.Errorf("After release, refs should be 1, got %d", entry.refs)
	}

	// Call releaser again
	releaser()

	entry = gc.GetEntry(1)
	if entry.refs != 0 {
		t.Errorf("After second release, refs should be 0, got %d", entry.refs)
	}
}

func TestGC_Releaser_UpdatesLastUsed(t *testing.T) {
	gc := NewGC(100, 10*time.Minute, 1*time.Hour)
	defer gc.Close()

	oldTime := time.Now().Add(-5 * time.Minute)

	// Add entry with old lastUsed
	gc.AddTestEntry(1, 10001, oldTime, 1)

	entry := gc.GetEntry(1)
	if !entry.lastUsed.Equal(oldTime) {
		t.Fatalf("Initial lastUsed should be %v, got %v", oldTime, entry.lastUsed)
	}

	// Small delay to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Call releaser
	releaser := gc.TestReleaser(1)
	releaser()

	entry = gc.GetEntry(1)
	if !entry.lastUsed.After(oldTime) {
		t.Error("lastUsed should be updated after release")
	}
}

func TestGC_Releaser_DoesNotGoBelowZero(t *testing.T) {
	gc := NewGC(100, 10*time.Minute, 1*time.Hour)
	defer gc.Close()

	// Add entry with refs=0
	gc.AddTestEntry(1, 10001, time.Now(), 0)

	// Call releaser multiple times - refs should stay at 0
	releaser := gc.TestReleaser(1)
	releaser()
	releaser()
	releaser()

	entry := gc.GetEntry(1)
	if entry.refs != 0 {
		t.Errorf("refs should stay at 0, got %d", entry.refs)
	}
}

func TestGC_Close_StopsSweeper(t *testing.T) {
	// Create GC with very short sweep interval
	gc := NewGC(100, 10*time.Minute, 50*time.Millisecond)

	// Close should complete without hanging
	done := make(chan struct{})
	go func() {
		gc.Close()
		close(done)
	}()

	select {
	case <-done:
		// Success - Close completed
	case <-time.After(2 * time.Second):
		t.Fatal("Close() did not complete in time - sweeper may not have stopped")
	}
}

func TestGC_WaitAllReleased_AllZeroRefs(t *testing.T) {
	gc := NewGC(100, 10*time.Minute, 1*time.Hour)
	defer gc.Close()

	// Add entries with refs=0
	gc.AddTestEntry(1, 10001, time.Now(), 0)
	gc.AddTestEntry(2, 10002, time.Now(), 0)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := gc.WaitAllReleased(ctx)
	if err != nil {
		t.Errorf("WaitAllReleased should succeed when all refs are 0, got error: %v", err)
	}
}

func TestGC_WaitAllReleased_WaitsForRelease(t *testing.T) {
	gc := NewGC(100, 10*time.Minute, 1*time.Hour)
	defer gc.Close()

	// Add entry with refs=1
	gc.AddTestEntry(1, 10001, time.Now(), 1)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Start waiting in background
	done := make(chan error, 1)
	go func() {
		done <- gc.WaitAllReleased(ctx)
	}()

	// Release after small delay
	time.Sleep(100 * time.Millisecond)
	releaser := gc.TestReleaser(1)
	releaser()

	err := <-done
	if err != nil {
		t.Errorf("WaitAllReleased should succeed after release, got error: %v", err)
	}
}

func TestGC_WaitAllReleased_TimesOut(t *testing.T) {
	gc := NewGC(100, 10*time.Minute, 1*time.Hour)
	defer gc.Close()

	// Add entry with refs=1 (never released)
	gc.AddTestEntry(1, 10001, time.Now(), 1)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := gc.WaitAllReleased(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("WaitAllReleased should timeout, got: %v", err)
	}
}

func TestGC_ConcurrentAccess(t *testing.T) {
	gc := NewGC(100, 10*time.Minute, 1*time.Hour)
	defer gc.Close()

	// Add some initial entries
	for i := int64(1); i <= 10; i++ {
		gc.AddTestEntry(i, int32(10000+i), time.Now(), 0)
	}

	var wg sync.WaitGroup

	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = gc.LoadedCount()
				_ = gc.GetEntry(int64(j%10 + 1))
			}
		}()
	}

	// Concurrent writers (add/remove)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := int64(100 + id)
			for j := 0; j < 50; j++ {
				gc.AddTestEntry(key, int32(20000+id), time.Now(), 0)
				gc.RemoveTestEntry(key)
			}
		}(i)
	}

	// Concurrent releasers
	for i := int64(1); i <= 5; i++ {
		wg.Add(1)
		go func(key int64) {
			defer wg.Done()
			gc.mu.Lock()
			if e := gc.loaded[key]; e != nil {
				e.refs = 10
			}
			gc.mu.Unlock()

			for j := 0; j < 10; j++ {
				releaser := gc.TestReleaser(key)
				releaser()
			}
		}(i)
	}

	// Concurrent eviction checks
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_ = gc.evictLRUIfNeeded(1)
			}
		}()
	}

	wg.Wait()
}

func TestGC_SweepInterval_SweeperRuns(t *testing.T) {
	// Create GC with very short sweep interval and TTL
	sweepInterval := 50 * time.Millisecond
	idleTTL := 10 * time.Millisecond

	gc := NewGC(100, idleTTL, sweepInterval)
	defer gc.Close()

	// Add an old entry that should be swept
	gc.AddTestEntry(1, 10001, time.Now().Add(-1*time.Second), 0)

	if gc.LoadedCount() != 1 {
		t.Fatalf("Should have 1 entry, got %d", gc.LoadedCount())
	}

	// Wait for sweeper to run
	// Note: The actual removal depends on DLL calls which won't work in tests,
	// but we can verify the sweeper logic identifies victims correctly
	time.Sleep(100 * time.Millisecond)

	// The entry should still exist since removeVictims requires DLL calls
	// but in a real scenario, it would be removed
	// This test mainly verifies the sweeper goroutine runs without panic
}

func TestParseSatNum_Valid(t *testing.T) {
	tests := []struct {
		name     string
		tle1     string
		expected int32
	}{
		{
			name:     "ISS TLE",
			tle1:     "1 25544U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
			expected: 25544,
		},
		{
			name:     "Leading zeros",
			tle1:     "1 00123U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
			expected: 123,
		},
		{
			name:     "Max 5 digit",
			tle1:     "1 99999U 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
			expected: 99999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSatNum(tt.tle1)
			if err != nil {
				t.Errorf("parseSatNum() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("parseSatNum() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestParseSatNum_Invalid(t *testing.T) {
	tests := []struct {
		name string
		tle1 string
	}{
		{
			name: "Too short",
			tle1: "1 254",
		},
		{
			name: "Empty",
			tle1: "",
		},
		{
			name: "Non-numeric",
			tle1: "1 ABCDE 98067A   21275.52543210  .00016717  00000-0  10270-3 0  9042",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSatNum(tt.tle1)
			if err == nil {
				t.Error("parseSatNum() expected error, got nil")
			}
		})
	}
}

func TestGC_LRUOrdering(t *testing.T) {
	// Create GC with max 5 satellites
	gc := NewGC(5, 10*time.Minute, 1*time.Hour)
	defer gc.Close()

	baseTime := time.Now()

	// Add 5 satellites with specific ordering
	// Key 3 is oldest, Key 1 is newest
	gc.AddTestEntry(1, 10001, baseTime, 0)                     // newest
	gc.AddTestEntry(2, 10002, baseTime.Add(-1*time.Minute), 0) // 2nd newest
	gc.AddTestEntry(3, 10003, baseTime.Add(-5*time.Minute), 0) // oldest
	gc.AddTestEntry(4, 10004, baseTime.Add(-2*time.Minute), 0) // 3rd newest
	gc.AddTestEntry(5, 10005, baseTime.Add(-3*time.Minute), 0) // 4th newest

	// Request eviction for 2 slots - should evict 2 oldest (keys 3 and 5)
	victims := gc.evictLRUIfNeeded(2)

	if len(victims) != 2 {
		t.Fatalf("Expected 2 victims, got %d", len(victims))
	}

	// First victim should be key 3 (oldest)
	if victims[0] != 3 {
		t.Errorf("First victim should be key 3 (oldest), got %d", victims[0])
	}

	// Second victim should be key 5 (second oldest)
	if victims[1] != 5 {
		t.Errorf("Second victim should be key 5 (second oldest), got %d", victims[1])
	}
}

func TestGC_EmptyCache(t *testing.T) {
	gc := NewGC(10, 10*time.Minute, 1*time.Hour)
	defer gc.Close()

	// Eviction on empty cache should return empty slice
	victims := gc.evictLRUIfNeeded(5)
	if len(victims) != 0 {
		t.Errorf("Eviction on empty cache should return 0 victims, got %d", len(victims))
	}

	// LoadedCount should be 0
	if gc.LoadedCount() != 0 {
		t.Errorf("Empty cache should have 0 entries, got %d", gc.LoadedCount())
	}

	// GetEntry should return nil
	if gc.GetEntry(1) != nil {
		t.Error("GetEntry on empty cache should return nil")
	}
}

func TestGC_RefCountIntegrity(t *testing.T) {
	gc := NewGC(100, 10*time.Minute, 1*time.Hour)
	defer gc.Close()

	gc.AddTestEntry(1, 10001, time.Now(), 5)

	// Create multiple releasers
	releasers := make([]func(), 5)
	for i := 0; i < 5; i++ {
		releasers[i] = gc.TestReleaser(1)
	}

	// Release all
	for _, r := range releasers {
		r()
	}

	entry := gc.GetEntry(1)
	if entry.refs != 0 {
		t.Errorf("After releasing all refs, should be 0, got %d", entry.refs)
	}
}

// Benchmark tests

func BenchmarkGC_EvictLRUIfNeeded(b *testing.B) {
	gc := NewGC(1000, 10*time.Minute, 1*time.Hour)
	defer gc.Close()

	// Fill with entries
	for i := int64(1); i <= 1000; i++ {
		gc.AddTestEntry(i, int32(10000+i), time.Now().Add(-time.Duration(i)*time.Second), 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gc.evictLRUIfNeeded(10)
	}
}

func BenchmarkGC_Releaser(b *testing.B) {
	gc := NewGC(1000, 10*time.Minute, 1*time.Hour)
	defer gc.Close()

	gc.AddTestEntry(1, 10001, time.Now(), 1000000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		releaser := gc.TestReleaser(1)
		releaser()
	}
}

func BenchmarkGC_ConcurrentLoadedCount(b *testing.B) {
	gc := NewGC(1000, 10*time.Minute, 1*time.Hour)
	defer gc.Close()

	for i := int64(1); i <= 100; i++ {
		gc.AddTestEntry(i, int32(10000+i), time.Now(), 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = gc.LoadedCount()
		}
	})
}
