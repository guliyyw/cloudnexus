package system

import (
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var startTime = time.Now()

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

		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		c.JSON(200, gin.H{
			"status":     "ok",
			"service":    service,
			"uptime":     time.Since(startTime).String(),
			"go_version": runtime.Version(),
			"goroutines": runtime.NumGoroutine(),
			"memory_mb":  memStats.Alloc / 1024 / 1024,
			"components": components,
		})
	}
}
