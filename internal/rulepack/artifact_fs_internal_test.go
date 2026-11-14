package rulepack

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRegularFileReadStaysBoundedWhileFileGrows(t *testing.T) {
	target := filepath.Join(t.TempDir(), "partial.md")
	const limit = int64(64)
	if err := os.WriteFile(target, []byte("small"), 0o600); err != nil {
		t.Fatal(err)
	}
	writer, err := os.OpenFile(target, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	stop, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		defer writer.Close()
		large := bytes.Repeat([]byte("x"), 4096)
		for {
			select {
			case <-stop:
				return
			default:
			}
			if writer.Truncate(5) != nil {
				return
			}
			runtime.Gosched()
			if _, err := writer.WriteAt(large, 0); err != nil {
				return
			}
			runtime.Gosched()
		}
	}()
	defer func() { close(stop); <-done }()
	for attempt := 0; attempt < 10_000; attempt++ {
		data, _, _ := readRegularFileLimited(target, "partial.md", limit)
		if int64(len(data)) > limit {
			t.Fatalf("runtime reader exceeded its ceiling: read=%d limit=%d", len(data), limit)
		}
		runtime.Gosched()
	}
}
