package system

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var startTime = time.Now()
var cpuSampleMu sync.Mutex
var lastProcJiffies uint64
var lastTotalJiffies uint64

// ComponentCheck is a function that returns a component name and its status ("ok" or "error: ...").
type ComponentCheck func() (name, status string)

// HealthzHandler returns a gin handler that responds with detailed health information.
// Checks are run in parallel.
func HealthzHandler(service string, checks ...ComponentCheck) gin.HandlerFunc {
	return func(c *gin.Context) {
		components := make(map[string]string, len(checks))

		if len(checks) > 0 {
			var wg sync.WaitGroup
			var mu sync.Mutex
			for _, check := range checks {
				wg.Add(1)
				go func(chk ComponentCheck) {
					defer wg.Done()
					name, status := chk()
					mu.Lock()
					components[name] = status
					mu.Unlock()
				}(check)
			}
			wg.Wait()
		}

		cpuPercent, memoryMB, memorySysMB := ProcessResourceUsage()

		c.JSON(200, gin.H{
			"status":        "ok",
			"service":       service,
			"version":       os.Getenv("SERVICE_VERSION"),
			"uptime":        time.Since(startTime).String(),
			"go_version":    runtime.Version(),
			"goroutines":    runtime.NumGoroutine(),
			"cpu_percent":   cpuPercent,
			"memory_mb":     memoryMB,
			"memory_sys_mb": memorySysMB,
			"components":    components,
		})
	}
}

// ProcessResourceUsage returns process CPU usage and Go runtime memory in MiB.
// It is shared by service-specific health handlers so every service reports
// resource data with the same units and sampling method.
func ProcessResourceUsage() (cpuPercent float64, memoryMB, memorySysMB uint64) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return sampleProcessCPUPercent(), memStats.Alloc / 1024 / 1024, memStats.Sys / 1024 / 1024
}

func sampleProcessCPUPercent() float64 {
	procJiffies, ok := readProcessJiffies()
	if !ok {
		return 0
	}
	totalJiffies, ok := readTotalCPUJiffies()
	if !ok {
		return 0
	}

	cpuSampleMu.Lock()
	defer cpuSampleMu.Unlock()

	if lastProcJiffies == 0 || lastTotalJiffies == 0 || totalJiffies <= lastTotalJiffies {
		lastProcJiffies = procJiffies
		lastTotalJiffies = totalJiffies
		return 0
	}

	procDelta := procJiffies - lastProcJiffies
	totalDelta := totalJiffies - lastTotalJiffies
	lastProcJiffies = procJiffies
	lastTotalJiffies = totalJiffies
	if totalDelta == 0 {
		return 0
	}

	percent := (float64(procDelta) / float64(totalDelta)) * float64(runtime.NumCPU()) * 100
	if percent < 0 {
		return 0
	}
	return percent
}

func readProcessJiffies() (uint64, bool) {
	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0, false
	}
	text := string(data)
	idx := strings.LastIndex(text, ")")
	if idx < 0 || idx+2 >= len(text) {
		return 0, false
	}
	fields := strings.Fields(text[idx+2:])
	if len(fields) <= 12 {
		return 0, false
	}
	utime, err1 := strconv.ParseUint(fields[11], 10, 64)
	stime, err2 := strconv.ParseUint(fields[12], 10, 64)
	if err1 != nil || err2 != nil {
		return 0, false
	}
	return utime + stime, true
}

func readTotalCPUJiffies() (uint64, bool) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, false
	}
	lines := strings.SplitN(string(data), "\n", 2)
	fields := strings.Fields(lines[0])
	if len(fields) < 2 || fields[0] != "cpu" {
		return 0, false
	}
	var total uint64
	for _, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return 0, false
		}
		total += value
	}
	return total, true
}
