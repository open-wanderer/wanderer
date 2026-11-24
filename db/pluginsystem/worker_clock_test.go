package pluginsystem

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// This minimal guest reads actual WASI realtime and monotonic clocks. Its
// exports return seconds and the low 32 bits of nanoseconds respectively;
// no guest compiler or external artifact is needed for the runtime test.
func workerClockGuest() []byte {
	leb := func(value uint64) []byte {
		encoded := []byte{}
		for {
			next := byte(value & 0x7f)
			value >>= 7
			if value != 0 {
				next |= 0x80
			}
			encoded = append(encoded, next)
			if value == 0 {
				return encoded
			}
		}
	}
	name := func(value string) []byte { return append(leb(uint64(len(value))), []byte(value)...) }
	wasm := []byte{0, 97, 115, 109, 1, 0, 0, 0}
	section := func(id byte, payload []byte) {
		wasm = append(wasm, id)
		wasm = append(wasm, leb(uint64(len(payload)))...)
		wasm = append(wasm, payload...)
	}
	section(1, []byte{2, 0x60, 3, 0x7f, 0x7e, 0x7f, 1, 0x7f, 0x60, 0, 1, 0x7f})
	imports := append([]byte{1}, name("wasi_snapshot_preview1")...)
	imports = append(imports, name("clock_time_get")...)
	imports = append(imports, 0, 0)
	section(2, imports)
	section(3, []byte{2, 1, 1})
	section(5, []byte{1, 0, 1})
	exports := append([]byte{3}, name("memory")...)
	exports = append(exports, 2, 0)
	exports = append(exports, name("walltime")...)
	exports = append(exports, 0, 1)
	exports = append(exports, name("nanotime")...)
	exports = append(exports, 0, 2)
	section(7, exports)
	wall := []byte{0, 0x41, 0, 0x42, 1, 0x41, 0, 0x10, 0, 0x1a, 0x41, 0, 0x29, 3, 0, 0x42}
	wall = append(wall, leb(1_000_000_000)...)
	wall = append(wall, 0x80, 0xa7, 0x0b)
	mono := []byte{0, 0x41, 1, 0x42, 1, 0x41, 8, 0x10, 0, 0x1a, 0x41, 8, 0x29, 3, 0, 0xa7, 0x0b}
	code := []byte{2}
	code = append(code, leb(uint64(len(wall)))...)
	code = append(code, wall...)
	code = append(code, leb(uint64(len(mono)))...)
	code = append(code, mono...)
	section(10, code)
	return wasm
}

func TestPluginWorkerProvidesCurrentWalltimeAndAdvancingMonotonicClock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clock.wasm")
	if err := os.WriteFile(path, workerClockGuest(), 0600); err != nil {
		t.Fatal(err)
	}
	worker := &pluginWorkerProcess{ctx: t.Context()}
	if err := worker.openPlugin(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = worker.instance.Close(t.Context()) })
	before := time.Now().Unix()
	wall, _, err := worker.instance.CallWithContext(t.Context(), "walltime", nil)
	after := time.Now().Unix()
	if err != nil {
		t.Fatal(err)
	}
	if int64(wall) < before-1 || int64(wall) > after+1 {
		t.Errorf("guest walltime=%d, current host interval=%d..%d", wall, before, after)
	}
	first, _, err := worker.instance.CallWithContext(t.Context(), "nanotime", nil)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	<-time.After(10 * time.Millisecond)
	second, _, err := worker.instance.CallWithContext(t.Context(), "nanotime", nil)
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(started)
	if delta := time.Duration(uint32(second - first)); delta < elapsed/2 {
		t.Errorf("guest monotonic clock advanced only %v during host wait of %v", delta, elapsed)
	}
}
