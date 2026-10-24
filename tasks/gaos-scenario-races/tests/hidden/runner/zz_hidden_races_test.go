package runner

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

// Hidden tests. The grader runs them with -race: a data race fails the test binary.

func sequentialRejects(exec func() (Done, error), calls int) int {
	n := 0
	for i := 0; i < calls; i++ {
		if _, err := exec(); err != nil {
			n++
		}
	}
	return n
}

func concurrentRejects(exec func() (Done, error), calls, workers int) int {
	var rejected int64
	var wg sync.WaitGroup
	jobs := make(chan struct{})
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				if _, err := exec(); err != nil {
					atomic.AddInt64(&rejected, 1)
				}
			}
		}()
	}
	for i := 0; i < calls; i++ {
		jobs <- struct{}{}
	}
	close(jobs)
	wg.Wait()
	return int(rejected)
}

func TestHiddenRateCountsLikeSequential(t *testing.T) {
	for _, r := range []int{1, 3, 10} {
		want := sequentialRejects(NewRate(Scenario{Rate: r}).Execute, 990)
		got := concurrentRejects(NewRate(Scenario{Rate: r}).Execute, 990, 48)
		if got != want {
			t.Fatalf("rate %d: %d rejections under concurrency, %d sequentially", r, got, want)
		}
	}
}

func TestHiddenLimitCountsLikeSequential(t *testing.T) {
	for _, l := range []int{1, 37, 400} {
		want := sequentialRejects(NewLimit(Scenario{Limit: l}).Execute, 500)
		got := concurrentRejects(NewLimit(Scenario{Limit: l}).Execute, 500, 48)
		if got != want {
			t.Fatalf("limit %d: %d rejections under concurrency, %d sequentially", l, got, want)
		}
	}
}

// From the project's fix: a "direct" action swaps the Method's scenario while other requests run.
func TestHiddenMethodDirectSwitchUnderConcurrency(t *testing.T) {
	g := &Runner{Scenario: map[string]*Scenario{
		"a": {Name: "a", Rate: 1, Ignore: Action{Direct: "b"}},
		"b": {Name: "b", Rate: 1, Ignore: Action{Direct: "a"}},
	}}
	g.resolveScenarios()
	m := &Method{Scenario: *g.Scenario["a"], runner: g}
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Execute()
		}()
	}
	wg.Wait()
}

func TestHiddenHandlerUnderConcurrentRequests(t *testing.T) {
	g := &Runner{Scenario: map[string]*Scenario{
		"open": {Name: "open", Rate: 2,
			Accept: Action{Status: 200, Result: Result{Type: "static", Content: map[string]interface{}{"s": "open"}}},
			Ignore: Action{Status: 429, Direct: "tight", Result: Result{Type: "static", Content: map[string]interface{}{"s": "busy"}}}},
		"tight": {Name: "tight", Limit: 1000, Rate: 1,
			Accept: Action{Status: 200, Direct: "open", Result: Result{Type: "static", Content: map[string]interface{}{"s": "tight"}}},
			Ignore: Action{Status: 429, Result: Result{Type: "static", Content: map[string]interface{}{"s": "busy"}}}},
	}}
	g.Metrics.endpointCallCounts = map[string]int{}
	g.resolveScenarios()
	m := &Method{Scenario: *g.Scenario["open"], runner: g}

	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()
	srv := &fasthttp.Server{Handler: m.Handler()}
	go func() { _ = srv.Serve(ln) }()
	var wg sync.WaitGroup
	var bad int64
	for w := 0; w < 32; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			c := &fasthttp.HostClient{Addr: "gaos", Dial: func(string) (net.Conn, error) { return ln.Dial() }}
			for i := 0; i < 12; i++ {
				req, res := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
				req.SetRequestURI(fmt.Sprintf("http://gaos/x?w=%d&i=%d", w, i))
				if err := c.DoTimeout(req, res, 5*time.Second); err != nil || (res.StatusCode() != 200 && res.StatusCode() != 429) {
					atomic.AddInt64(&bad, 1)
				}
				fasthttp.ReleaseRequest(req)
				fasthttp.ReleaseResponse(res)
			}
		}(w)
	}
	wg.Wait()
	if bad != 0 {
		t.Fatalf("%d requests failed or got an unexpected status", bad)
	}
}
