package metrics

import (
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
)

// Collector собирает метрики runtime и отдает их в формате Prometheus.
type Collector struct {
	gcPercent atomic.Int64
}

// NewCollector создает сборщик метрик с текущим значением GOGC.
func NewCollector(gcPercent int) *Collector {
	collector := &Collector{}
	collector.gcPercent.Store(int64(gcPercent))

	return collector
}

// SetGCPercent сохраняет текущее значение GOGC для вывода в метриках.
func (c *Collector) SetGCPercent(percent int) {
	c.gcPercent.Store(int64(percent))
}

// GCPercent возвращает сохраненное значение GOGC.
func (c *Collector) GCPercent() int {
	return int(c.gcPercent.Load())
}

// Prometheus собирает runtime.MemStats и формирует текст Prometheus exposition format.
func (c *Collector) Prometheus() string {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	var b strings.Builder

	writeGauge(&b, "go_mem_alloc_bytes", "Bytes of allocated heap objects.", mem.Alloc)
	writeCounter(&b, "go_mem_total_alloc_bytes", "Cumulative bytes allocated for heap objects.", mem.TotalAlloc)
	writeGauge(&b, "go_mem_sys_bytes", "Total bytes of memory obtained from the OS.", mem.Sys)
	writeGauge(&b, "go_heap_alloc_bytes", "Bytes of allocated heap objects.", mem.HeapAlloc)
	writeGauge(&b, "go_heap_sys_bytes", "Bytes of heap memory obtained from the OS.", mem.HeapSys)
	writeGauge(&b, "go_heap_idle_bytes", "Bytes in idle heap spans.", mem.HeapIdle)
	writeGauge(&b, "go_heap_inuse_bytes", "Bytes in in-use heap spans.", mem.HeapInuse)
	writeGauge(&b, "go_heap_released_bytes", "Bytes of physical memory returned to the OS.", mem.HeapReleased)
	writeGauge(&b, "go_heap_objects", "Number of allocated heap objects.", mem.HeapObjects)

	writeGauge(&b, "go_stack_inuse_bytes", "Bytes in stack spans.", mem.StackInuse)
	writeGauge(&b, "go_stack_sys_bytes", "Bytes of stack memory obtained from the OS.", mem.StackSys)
	writeGauge(&b, "go_mspan_inuse_bytes", "Bytes of allocated mspan structures.", mem.MSpanInuse)
	writeGauge(&b, "go_mcache_inuse_bytes", "Bytes of allocated mcache structures.", mem.MCacheInuse)

	writeGauge(&b, "go_next_gc_bytes", "Target heap size of the next GC cycle.", mem.NextGC)
	writeCounter(&b, "go_gc_cycles_total", "Cumulative number of completed GC cycles.", uint64(mem.NumGC))
	writeCounter(&b, "go_gc_pause_total_ns", "Cumulative nanoseconds in GC stop-the-world pauses.", mem.PauseTotalNs)
	writeGauge(&b, "go_gc_cpu_fraction", "Fraction of CPU time used by GC since the program started.", mem.GCCPUFraction)
	writeGauge(&b, "go_gc_last_timestamp_seconds", "Unix timestamp of the last completed GC cycle.", float64(mem.LastGC)/1e9)
	writeGauge(&b, "go_gc_last_pause_ns", "Nanoseconds in the latest GC pause.", lastPauseNs(mem))
	writeGauge(&b, "go_gc_percent", "Current GOGC value configured by the application.", c.GCPercent())

	writeGauge(&b, "go_goroutines", "Number of current goroutines.", runtime.NumGoroutine())
	writeBuildInfo(&b)

	return b.String()
}

func lastPauseNs(mem runtime.MemStats) uint64 {
	if mem.NumGC == 0 {
		return 0
	}

	return mem.PauseNs[(mem.NumGC+255)%256]
}

func writeGauge[T number](b *strings.Builder, name string, help string, value T) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s gauge\n", name)
	fmt.Fprintf(b, "%s %v\n\n", name, value)
}

func writeCounter[T number](b *strings.Builder, name string, help string, value T) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s counter\n", name)
	fmt.Fprintf(b, "%s %v\n\n", name, value)
}

func writeBuildInfo(b *strings.Builder) {
	fmt.Fprintln(b, "# HELP gc_mem_exporter_build_info Build information about the application runtime.")
	fmt.Fprintln(b, "# TYPE gc_mem_exporter_build_info gauge")
	fmt.Fprintf(
		b,
		"gc_mem_exporter_build_info{go_version=%q,go_os=%q,go_arch=%q} 1\n\n",
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	)
}

type number interface {
	~int | ~int64 | ~uint32 | ~uint64 | ~float64
}
