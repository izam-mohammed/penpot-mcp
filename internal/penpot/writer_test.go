package penpot

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testFile = "f0000000-0000-4000-8000-000000000001"

func TestBoxGeometry(t *testing.T) {
	// Fixture: docs/penpot-api-reference.md section 7.8, the minimal rectangle.
	const want = `{"x":100,"y":100,"width":200,"height":120,"rotation":0,` +
		`"selrect":{"x":100,"y":100,"width":200,"height":120,"x1":100,"y1":100,"x2":300,"y2":220},` +
		`"points":[{"x":100,"y":100},{"x":300,"y":100},{"x":300,"y":220},{"x":100,"y":220}],` +
		`"transform":{"a":1,"b":0,"c":0,"d":1,"e":0,"f":0},` +
		`"transformInverse":{"a":1,"b":0,"c":0,"d":1,"e":0,"f":0}}`
	got, err := json.Marshal(Box(100, 100, 200, 120))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("Box =\n%s\nwant\n%s", got, want)
	}
}

func TestWriterSerialises(t *testing.T) {
	f, srv := newFake(t)
	f.on("get-file", reply(fileReply(0, 0, true)))
	var inFlight, peak int32
	var revn int64
	var mu sync.Mutex
	f.on("update-file", func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&inFlight, 1)
		for {
			p := atomic.LoadInt32(&peak)
			if n <= p || atomic.CompareAndSwapInt32(&peak, p, n) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&inFlight, -1)
		mu.Lock()
		before := revn
		revn++
		mu.Unlock()
		reply(map[string]any{"revn": before, "lagged": []any{}})(w, r)
	})
	wr := NewWriter(testClient(t, srv))

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := wr.Apply(context.Background(), testFile, []map[string]any{{"type": "noop"}}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if peak != 1 {
		t.Errorf("peak concurrent update-file = %d, want 1", peak)
	}
	// Each call sends the revn the previous one left: 0..7.
	f.mu.Lock()
	defer f.mu.Unlock()
	seen := map[float64]bool{}
	for i, c := range f.calls {
		if c == "update-file" {
			seen[f.bodies[i]["revn"].(float64)] = true
		}
	}
	if len(seen) != 8 {
		t.Errorf("distinct revns sent = %v, want 0..7", seen)
	}
}

func TestWriterRetriesOnVernConflict(t *testing.T) {
	f, srv := newFake(t)
	gets := 0
	f.on("get-file", func(w http.ResponseWriter, r *http.Request) {
		gets++
		reply(fileReply(5, int64(gets), true))(w, r)
	})
	updates := 0
	f.on("update-file", func(w http.ResponseWriter, r *http.Request) {
		updates++
		if updates == 1 {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"type":"validation","code":"vern-conflict"}`))
			return
		}
		reply(map[string]any{"revn": 5})(w, r)
	})
	wr := NewWriter(testClient(t, srv))
	if err := wr.Apply(context.Background(), testFile, nil); err != nil {
		t.Fatal(err)
	}
	if gets != 2 || updates != 2 {
		t.Errorf("get-file %d, update-file %d; want 2 and 2", gets, updates)
	}
	if v := f.bodies[len(f.bodies)-1]["vern"].(float64); v != 2 {
		t.Errorf("retry sent vern %v, want the re-read 2", v)
	}
}

func TestWriterRefusesReadOnlyFile(t *testing.T) {
	f, srv := newFake(t)
	f.on("get-file", reply(fileReply(0, 0, false)))
	wr := NewWriter(testClient(t, srv))
	if err := wr.Apply(context.Background(), testFile, nil); err == nil {
		t.Fatal("want an error for a file the token cannot edit")
	}
	if f.callCount("update-file") != 0 {
		t.Error("sent update-file to a read-only file")
	}
}

func TestNewID(t *testing.T) {
	id := NewID()
	if len(id) != 36 || id[14] != '4' {
		t.Errorf("NewID = %q, want a v4 UUID", id)
	}
}
