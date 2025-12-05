// main executable.
package main

import (
	"os"
	"runtime/debug"

	"github.com/bluenviron/mediamtx/internal/core"
)

func main() {
	// Performance optimization: Increase GC threshold to reduce GC frequency
	// GOGC=300 means GC runs when heap grows to 300% (4x) of previous size
	// Trade-off: +10-15 MB memory for -0.5-1% additional CPU reduction
	// Profiling results (2025-12-04):
	// - GOGC=200 result: GC 0.72s (1.49% CPU), Memory 38.61 MB
	// - GOGC=300 target: GC < 0.5s (< 1% CPU), Memory ~50 MB
	// - Previous GOGC=200 achieved -70.5% GC reduction, expecting further -30-50% reduction
	debug.SetGCPercent(300)

	s, ok := core.New(os.Args[1:])
	if !ok {
		os.Exit(1)
	}
	s.Wait()
}
