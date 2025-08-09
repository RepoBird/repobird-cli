# Task 04h: Performance Optimization

## Overview
Optimize RepoBird CLI performance to achieve <100ms startup time, <20MB binary size, and minimal resource usage while maintaining functionality.

## Background Research

### Performance Best Practices for Go CLIs
Based on industry standards:
- **CPU Profiling** - Use pprof and Profile-Guided Optimization (PGO)
- **Memory Optimization** - Minimize allocations, use sync.Pool, prevent leaks
- **Binary Size** - Strip debug info, remove unused dependencies
- **Fast Startup** - Defer heavy operations, lazy loading
- **Smart Caching** - Cache hot data only, expire cold items
- **Connection Pooling** - Reuse network connections
- **Lazy Loading** - Load resources only on demand

## Implementation Tasks

### 1. Performance Profiling Setup
- [ ] Add profiling support:
  ```go
  // internal/profiling/profiler.go
  package profiling
  
  import (
      "os"
      "runtime/pprof"
      "runtime/trace"
  )
  
  func StartCPUProfile(file string) (*os.File, error) {
      f, err := os.Create(file)
      if err != nil {
          return nil, err
      }
      pprof.StartCPUProfile(f)
      return f, nil
  }
  
  func WriteMemProfile(file string) error {
      f, err := os.Create(file)
      if err != nil {
          return err
      }
      defer f.Close()
      return pprof.WriteHeapProfile(f)
  }
  ```
- [ ] Add `--profile` flag to commands
- [ ] Create benchmark suite
- [ ] Set up continuous profiling in CI
- [ ] Generate flame graphs

### 2. Startup Time Optimization
- [ ] Profile startup path:
  ```bash
  go build -o repobird ./cmd/repobird
  time ./repobird version  # Measure cold start
  go tool pprof -http=:8080 cpu.prof
  ```
- [ ] Defer heavy initialization:
  ```go
  // Lazy load configuration
  var (
      configOnce sync.Once
      config     *Config
  )
  
  func GetConfig() *Config {
      configOnce.Do(func() {
          config = loadConfig()
      })
      return config
  }
  ```
- [ ] Remove startup dependencies
- [ ] Parallelize independent init tasks
- [ ] Optimize import graph
- [ ] Measure and target <50ms for simple commands

### 3. Binary Size Reduction
- [ ] Configure build flags:
  ```makefile
  LDFLAGS := -ldflags "-s -w \
      -X main.version=$(VERSION) \
      -extldflags '-static'"
  
  build-small:
      CGO_ENABLED=0 go build $(LDFLAGS) \
          -trimpath \
          -o repobird \
          ./cmd/repobird
      
      # Optional: UPX compression
      upx --best --lzma repobird
  ```
- [ ] Audit dependencies:
  ```bash
  go mod graph | grep -v '@'
  go mod why -m <module>
  go list -m all | wc -l  # Count dependencies
  ```
- [ ] Replace heavy dependencies
- [ ] Use build tags to exclude features
- [ ] Remove embedded assets when possible
- [ ] Target <15MB uncompressed, <5MB with UPX

### 4. Memory Optimization
- [ ] Implement object pooling:
  ```go
  // internal/pool/buffers.go
  var bufferPool = sync.Pool{
      New: func() interface{} {
          return &bytes.Buffer{}
      },
  }
  
  func GetBuffer() *bytes.Buffer {
      return bufferPool.Get().(*bytes.Buffer)
  }
  
  func PutBuffer(buf *bytes.Buffer) {
      buf.Reset()
      bufferPool.Put(buf)
  }
  ```
- [ ] Reduce allocations in hot paths
- [ ] Use value types over pointers where appropriate
- [ ] Implement string interning for repeated strings
- [ ] Profile and fix memory leaks
- [ ] Target <20MB heap usage

### 5. Caching Strategy
- [ ] Implement multi-level cache:
  ```go
  // internal/cache/cache.go
  type Cache struct {
      memory *MemoryCache  // Hot data, TTL: 5 min
      disk   *DiskCache    // Warm data, TTL: 1 hour
  }
  
  type MemoryCache struct {
      mu    sync.RWMutex
      items map[string]*CacheItem
      maxSize int
  }
  
  type CacheItem struct {
      Value     interface{}
      ExpiresAt time.Time
      Size      int
  }
  ```
- [ ] Cache API responses
- [ ] Cache configuration values
- [ ] Cache git information
- [ ] Implement cache eviction (LRU)
- [ ] Add cache statistics

### 6. Connection Pooling
- [ ] Configure HTTP client:
  ```go
  // internal/api/client.go
  var httpClient = &http.Client{
      Timeout: 30 * time.Second,
      Transport: &http.Transport{
          MaxIdleConns:        100,
          MaxIdleConnsPerHost: 10,
          IdleConnTimeout:     90 * time.Second,
          DisableCompression:  false,
          DisableKeepAlives:   false,
      },
  }
  ```
- [ ] Implement connection reuse
- [ ] Add connection pool metrics
- [ ] Configure optimal pool sizes
- [ ] Handle connection lifecycle

### 7. Lazy Loading Implementation
- [ ] Defer plugin loading:
  ```go
  // internal/plugins/loader.go
  type PluginLoader struct {
      mu      sync.RWMutex
      plugins map[string]*Plugin
      loaded  map[string]bool
  }
  
  func (l *PluginLoader) Get(name string) (*Plugin, error) {
      l.mu.RLock()
      if l.loaded[name] {
          defer l.mu.RUnlock()
          return l.plugins[name], nil
      }
      l.mu.RUnlock()
      
      // Load plugin
      l.mu.Lock()
      defer l.mu.Unlock()
      if !l.loaded[name] {
          plugin, err := loadPlugin(name)
          if err != nil {
              return nil, err
          }
          l.plugins[name] = plugin
          l.loaded[name] = true
      }
      return l.plugins[name], nil
  }
  ```
- [ ] Lazy load configuration files
- [ ] Defer TUI initialization
- [ ] Load commands on demand
- [ ] Implement progressive enhancement

### 8. JSON/YAML Optimization
- [ ] Use efficient parsers:
  ```go
  // Use json.Decoder for streaming
  decoder := json.NewDecoder(reader)
  decoder.UseNumber() // Avoid float64 conversion
  
  // Use jsoniter for speed
  import jsoniter "github.com/json-iterator/go"
  var json = jsoniter.ConfigCompatibleWithStandardLibrary
  ```
- [ ] Implement schema validation caching
- [ ] Use code generation for marshaling
- [ ] Optimize struct tags
- [ ] Benchmark different libraries

### 9. Concurrency Optimization
- [ ] Limit goroutine creation:
  ```go
  // Use worker pool pattern
  type WorkerPool struct {
      workers int
      tasks   chan Task
      wg      sync.WaitGroup
  }
  
  func (p *WorkerPool) Start() {
      for i := 0; i < p.workers; i++ {
          p.wg.Add(1)
          go p.worker()
      }
  }
  ```
- [ ] Use context for cancellation
- [ ] Implement graceful shutdown
- [ ] Avoid goroutine leaks
- [ ] Profile concurrent operations

### 10. Profile-Guided Optimization (PGO)
- [ ] Collect production profiles:
  ```bash
  # Collect profiles from real usage
  ./repobird --profile=cpu.prof run task.json
  ./repobird --profile=cpu2.prof status --follow
  
  # Merge profiles
  go tool pprof -proto cpu.prof cpu2.prof > merged.pprof
  
  # Build with PGO (Go 1.21+)
  go build -pgo=merged.pprof -o repobird ./cmd/repobird
  ```
- [ ] Automate profile collection
- [ ] Regular PGO rebuilds
- [ ] Measure PGO improvements

## Performance Benchmarks

### Startup Time Targets
| Command | Target | Maximum |
|---------|--------|---------|
| `repobird version` | 20ms | 50ms |
| `repobird --help` | 30ms | 70ms |
| `repobird status` | 50ms | 100ms |
| `repobird run` | 70ms | 150ms |

### Memory Usage Targets
| Scenario | Target | Maximum |
|----------|--------|---------|
| Idle | 5MB | 10MB |
| Simple command | 10MB | 20MB |
| TUI mode | 20MB | 40MB |
| Large response | 30MB | 50MB |

### Binary Size Targets
| Build Type | Target | Maximum |
|------------|--------|---------|
| Debug | 25MB | 35MB |
| Release | 15MB | 20MB |
| UPX Compressed | 5MB | 8MB |

## Benchmark Suite

```go
// benchmarks/startup_test.go
func BenchmarkStartup(b *testing.B) {
    for i := 0; i < b.N; i++ {
        cmd := exec.Command("./repobird", "version")
        cmd.Run()
    }
}

func BenchmarkJSONParsing(b *testing.B) {
    data := []byte(`{"prompt": "test", "repository": "org/repo"}`)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        var task Task
        json.Unmarshal(data, &task)
    }
}

func BenchmarkAPICall(b *testing.B) {
    client := NewAPIClient()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        client.GetStatus("test-id")
    }
}
```

## Monitoring & Metrics

```go
// internal/metrics/collector.go
type Metrics struct {
    StartupTime     time.Duration
    MemoryUsage     uint64
    GoroutineCount  int
    APICallDuration time.Duration
    CacheHitRate    float64
}

func CollectMetrics() *Metrics {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    return &Metrics{
        MemoryUsage:    m.Alloc,
        GoroutineCount: runtime.NumGoroutine(),
    }
}
```

## Performance Testing Commands

```bash
# Run benchmarks
go test -bench=. -benchmem ./...

# Profile CPU
go test -cpuprofile=cpu.prof -bench=.

# Profile Memory
go test -memprofile=mem.prof -bench=.

# Analyze profiles
go tool pprof -http=:8080 cpu.prof
go tool pprof -http=:8081 mem.prof

# Trace execution
go test -trace=trace.out -bench=.
go tool trace trace.out

# Check binary size
go build -ldflags="-s -w" -o repobird ./cmd/repobird
ls -lh repobird
go tool nm -size -sort size repobird | head -20

# Memory usage
/usr/bin/time -v ./repobird status

# Startup time
hyperfine --warmup 3 './repobird version'
```

## Success Metrics
- Startup time < 100ms for all commands
- Binary size < 20MB (uncompressed)
- Memory usage < 20MB for typical operations
- Zero memory leaks detected
- CPU usage < 5% idle
- Cache hit rate > 80%

## Dependencies for Optimization
- `github.com/json-iterator/go` - Fast JSON parsing
- `github.com/klauspost/compress` - Better compression
- `github.com/valyala/fasthttp` - Fast HTTP (if needed)
- Standard library preferred for most operations

## References
- [Go Performance Optimization](https://go.dev/doc/pgo)
- [Profiling Go Programs](https://blog.golang.org/pprof)
- [High Performance Go](https://dave.cheney.net/high-performance-go-workshop/dotgo-paris.html)
- [Reducing Go Binary Size](https://blog.filippo.io/shrink-your-go-binaries-with-this-one-weird-trick/)