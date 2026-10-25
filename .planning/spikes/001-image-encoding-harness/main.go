// Spike 001/002 harness: decode → orient → resize ladder → encode, CGO-free.
//
//	bench   run one configuration in this process and print a JSON result line
//	matrix  run every configuration as a child process (clean peak RSS each) and print a table
//	fixtures  derive extra fixtures (WebP route-preview stand-in) from fixtures/12mp.jpg
//
// Build for a Pi: CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags nodynamic -o bin/imgbench-arm64 .
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gen2brain/jpegli"
	"github.com/gen2brain/webp"
	"github.com/rwcarlsen/goexif/exif"
)

type variant struct {
	Width int     `json:"width"`
	Bytes int     `json:"bytes"`
	SSIM  float64 `json:"ssim,omitempty"`
}

type result struct {
	Input      string    `json:"input"`
	InputBytes int       `json:"input_bytes"`
	SourceDims string    `json:"source_dims"`
	DecodedDim string    `json:"decoded_dims"`
	Decoder    string    `json:"decoder"`
	Encoder    string    `json:"encoder"`
	Quality    int       `json:"quality"`
	Procs      int       `json:"procs"`
	Runs       int       `json:"runs"`
	DecodeMs   float64   `json:"decode_ms"`
	ResizeMs   float64   `json:"resize_ms"`
	EncodeMs   float64   `json:"encode_ms"`
	TotalMs    float64   `json:"total_ms"`
	PeakRSSMB  float64   `json:"peak_rss_mb"`
	Variants   []variant `json:"variants"`
	Exif       string    `json:"exif"`
	Error      string    `json:"error,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: imgbench bench|matrix|fixtures [flags]")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "bench":
		benchCmd(os.Args[2:])
	case "matrix":
		matrixCmd(os.Args[2:])
	case "fixtures":
		fixturesCmd()
	default:
		fmt.Fprintln(os.Stderr, "unknown command", os.Args[1])
		os.Exit(2)
	}
}

func benchCmd(args []string) {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	in := fs.String("in", "fixtures/24mp.jpg", "input file")
	decoder := fs.String("decoder", "std", "std | jpegli-scaled")
	encoder := fs.String("encoder", "jpeg", "jpeg | webp | jpegli")
	quality := fs.Int("q", 80, "quality")
	ladder := fs.String("ladder", "2048,1280,800,400", "widths, largest first")
	runs := fs.Int("runs", 3, "timed runs (after one warm-up)")
	out := fs.String("out", "", "write the last run's variants here")
	memLimitMB := fs.Int("memlimit", 0, "GOMEMLIMIT in MB (0 = unset)")
	fs.Parse(args)

	if *memLimitMB > 0 {
		debug.SetMemoryLimit(int64(*memLimitMB) << 20)
	}

	widths := parseWidths(*ladder)
	data, err := os.ReadFile(*in)
	must(err)

	r := result{Input: filepath.Base(*in), InputBytes: len(data), Decoder: *decoder, Encoder: *encoder,
		Quality: *quality, Procs: runtime.GOMAXPROCS(0), Runs: *runs}
	r.Exif = exifSummary(data)

	var dec, rsz, enc time.Duration
	for i := 0; i <= *runs; i++ {
		d, z, e, vs, srcDims, decDims, err := process(data, *decoder, *encoder, *quality, widths, i == *runs, *out)
		if err != nil {
			r.Error = err.Error()
			break
		}
		r.SourceDims, r.DecodedDim, r.Variants = srcDims, decDims, vs
		if i == 0 {
			continue // warm-up: page faults, first module instantiation
		}
		dec += d
		rsz += z
		enc += e
	}
	n := float64(*runs)
	r.DecodeMs = ms(dec) / n
	r.ResizeMs = ms(rsz) / n
	r.EncodeMs = ms(enc) / n
	r.TotalMs = r.DecodeMs + r.ResizeMs + r.EncodeMs
	r.PeakRSSMB = peakRSSMB()
	must(json.NewEncoder(os.Stdout).Encode(r))
}

func process(data []byte, decoder, encoder string, quality int, widths []int, keep bool, outDir string) (dec, rsz, enc time.Duration, vs []variant, srcDims, decDims string, err error) {
	// Never go through the image.Decode registry: importing jpegli replaces the
	// "jpeg" format with jpegli, whose DecodeConfig panics on camera JPEGs.
	format := "jpeg"
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		if c, werr := webp.DecodeConfig(bytes.NewReader(data)); werr == nil {
			cfg, format, err = c, "webp", nil
		} else {
			return
		}
	}
	srcDims = fmt.Sprintf("%dx%d", cfg.Width, cfg.Height)

	t := time.Now()
	var img image.Image
	switch {
	case format == "webp":
		img, err = webp.Decode(bytes.NewReader(data))
	case decoder == "jpegli-scaled" && format == "jpeg":
		// Ask for the smallest N/8 scale that still covers the largest ladder width.
		long := max(cfg.Width, cfg.Height)
		target := widths[0]
		if long <= target {
			img, err = jpegli.Decode(bytes.NewReader(data))
		} else {
			tw := cfg.Width * target / long
			th := cfg.Height * target / long
			img, err = jpegli.DecodeWithOptions(bytes.NewReader(data), &jpegli.DecodingOptions{ScaleTarget: image.Rect(0, 0, tw, th), FancyUpsampling: true})
		}
	default:
		img, err = jpeg.Decode(bytes.NewReader(data))
	}
	if err != nil {
		return
	}
	img = orient(img, exifOrientation(data))
	dec = time.Since(t)
	decDims = fmt.Sprintf("%dx%d", img.Bounds().Dx(), img.Bounds().Dy())

	// Cascade: each variant is resized from the previous (larger) one, never upscaled.
	t = time.Now()
	resized := make([]image.Image, len(widths))
	src := img
	for i, w := range widths {
		// The ladder caps the long edge, so portrait photos come out at the same pixel budget.
		b := src.Bounds()
		if max(b.Dx(), b.Dy()) > w {
			if b.Dx() >= b.Dy() {
				src = imaging.Resize(src, w, 0, imaging.CatmullRom)
			} else {
				src = imaging.Resize(src, 0, w, imaging.CatmullRom)
			}
		}
		resized[i] = src
	}
	rsz = time.Since(t)

	for i, v := range resized {
		t = time.Now()
		var buf bytes.Buffer
		switch encoder {
		case "jpeg":
			err = jpeg.Encode(&buf, v, &jpeg.Options{Quality: quality})
		case "webp":
			err = webp.Encode(&buf, v, webp.Options{Quality: quality, Method: 4})
		case "png":
			err = png.Encode(&buf, v)
		case "jpegli":
			// Every field must be set: zero values disable optimized coding,
			// adaptive quantization and select 4:4:4.
			err = jpegli.Encode(&buf, v, &jpegli.EncodingOptions{Quality: quality, ChromaSubsampling: image.YCbCrSubsampleRatio420,
				ProgressiveLevel: 2, OptimizeCoding: true, AdaptiveQuantization: true, DCTMethod: jpegli.DefaultDCTMethod})
		default:
			err = fmt.Errorf("unknown encoder %q", encoder)
		}
		enc += time.Since(t)
		if err != nil {
			return
		}
		vs = append(vs, variant{Width: v.Bounds().Dx(), Bytes: buf.Len()})
		if keep {
			// Untimed: decode the output again and score it against the uncompressed resize.
			var back image.Image
			if encoder == "webp" {
				back, err = webp.Decode(bytes.NewReader(buf.Bytes()))
			} else if encoder == "png" {
				back, err = png.Decode(bytes.NewReader(buf.Bytes()))
			} else {
				back, err = jpeg.Decode(bytes.NewReader(buf.Bytes()))
			}
			if err != nil {
				return
			}
			vs[len(vs)-1].SSIM = ssim(v, back)
		}
		if keep && outDir != "" {
			ext := map[string]string{"jpeg": "jpg", "webp": "webp", "jpegli": "jpg", "png": "png"}[encoder]
			name := fmt.Sprintf("%s__%d.%s", encoder, widths[i], ext)
			_ = os.MkdirAll(outDir, 0o755)
			must(os.WriteFile(filepath.Join(outDir, name), buf.Bytes(), 0o644))
		}
	}
	return
}

func exifOrientation(data []byte) int {
	x, err := exif.Decode(bytes.NewReader(data))
	if err != nil {
		return 1
	}
	tag, err := x.Get(exif.Orientation)
	if err != nil {
		return 1
	}
	o, err := tag.Int(0)
	if err != nil {
		return 1
	}
	return o
}

func exifSummary(data []byte) string {
	x, err := exif.Decode(bytes.NewReader(data))
	if err != nil {
		return "none"
	}
	parts := []string{fmt.Sprintf("orientation=%d", exifOrientation(data))}
	if lat, lon, err := x.LatLong(); err == nil {
		parts = append(parts, fmt.Sprintf("gps=%.5f,%.5f", lat, lon))
	} else {
		parts = append(parts, "gps=none")
	}
	if tm, err := x.DateTime(); err == nil {
		parts = append(parts, "taken="+tm.Format(time.RFC3339))
	}
	return strings.Join(parts, " ")
}

func orient(img image.Image, o int) image.Image {
	switch o {
	case 2:
		return imaging.FlipH(img)
	case 3:
		return imaging.Rotate180(img)
	case 4:
		return imaging.FlipV(img)
	case 5:
		return imaging.Transpose(img)
	case 6:
		return imaging.Rotate270(img)
	case 7:
		return imaging.Transverse(img)
	case 8:
		return imaging.Rotate90(img)
	}
	return img
}

type config struct {
	in, decoder, encoder string
	procs                int
}

func matrixCmd(args []string) {
	fs := flag.NewFlagSet("matrix", flag.ExitOnError)
	inputs := fs.String("in", "fixtures/12mp.jpg,fixtures/24mp.jpg,fixtures/route.webp", "inputs")
	procsList := fs.String("procs", "1,0", "GOMAXPROCS values (0 = all cores)")
	runs := fs.Int("runs", 3, "timed runs per config")
	quality := fs.Int("q", 80, "quality")
	out := fs.String("out", "out", "variant output root")
	jsonOut := fs.String("json", "", "also write all results as JSON lines here")
	minAvailMB := fs.Int("minavail", 0, "abort before a run if the host's MemAvailable is below this (Linux only, 0 = off)")
	fs.Parse(args)

	self, err := os.Executable()
	must(err)

	var configs []config
	for _, in := range strings.Split(*inputs, ",") {
		if _, err := os.Stat(in); err != nil {
			fmt.Fprintln(os.Stderr, "skip missing", in)
			continue
		}
		for _, p := range parseWidths(*procsList) {
			for _, d := range []string{"std", "jpegli-scaled"} {
				if strings.HasSuffix(in, ".webp") && d != "std" {
					continue
				}
				for _, e := range []string{"jpeg", "webp", "jpegli"} {
					configs = append(configs, config{in, d, e, p})
				}
			}
		}
	}

	var results []result
	var jsonLines bytes.Buffer
	for _, c := range configs {
		if *minAvailMB > 0 {
			if avail, ok := memAvailableMB(); ok && avail < *minAvailMB {
				fmt.Fprintf(os.Stderr, "ABORT: MemAvailable %d MB < %d MB, stopping before %s %s %s\n", avail, *minAvailMB, c.in, c.decoder, c.encoder)
				break
			}
		}
		cmd := exec.Command(self, "bench", "-in", c.in, "-decoder", c.decoder, "-encoder", c.encoder,
			"-q", strconv.Itoa(*quality), "-runs", strconv.Itoa(*runs),
			"-out", filepath.Join(*out, strings.TrimSuffix(filepath.Base(c.in), filepath.Ext(c.in))+"__"+c.decoder))
		cmd.Env = os.Environ()
		if c.procs > 0 {
			cmd.Env = append(cmd.Env, "GOMAXPROCS="+strconv.Itoa(c.procs))
		}
		cmd.Stderr = os.Stderr
		b, err := cmd.Output()
		var r result
		if err != nil || json.Unmarshal(b, &r) != nil {
			r = result{Input: c.in, Decoder: c.decoder, Encoder: c.encoder, Error: fmt.Sprint(err)}
		}
		jsonLines.Write(b)
		results = append(results, r)
		fmt.Fprintf(os.Stderr, "done %s %s %s procs=%d\n", r.Input, c.decoder, c.encoder, r.Procs)
	}
	if *jsonOut != "" {
		must(os.WriteFile(*jsonOut, jsonLines.Bytes(), 0o644))
	}

	sort.SliceStable(results, func(i, j int) bool { return results[i].Input < results[j].Input })
	fmt.Printf("host: %s/%s, cpus=%d, q=%d, ladder 2048/1280/800/400 (long edge), mean of %d runs after warm-up\n\n",
		runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), *quality, *runs)
	fmt.Println("| input | src | decoder | decoded | encoder | procs | decode ms | resize ms | encode ms | total ms | peak RSS MB | KB 2048/1280/800/400 | SSIM 2048/1280/800/400 |")
	fmt.Println("|---|---|---|---|---|---|---|---|---|---|---|---|---|")
	for _, r := range results {
		if r.Error != "" {
			fmt.Printf("| %s | | %s | | %s | | ERROR %s |\n", r.Input, r.Decoder, r.Encoder, r.Error)
			continue
		}
		var kb, ss []string
		for _, v := range r.Variants {
			kb = append(kb, strconv.Itoa((v.Bytes+512)/1024))
			ss = append(ss, fmt.Sprintf("%.4f", v.SSIM))
		}
		fmt.Printf("| %s | %s | %s | %s | %s | %d | %.0f | %.0f | %.0f | **%.0f** | %.0f | %s | %s |\n",
			r.Input, r.SourceDims, r.Decoder, r.DecodedDim, r.Encoder, r.Procs, r.DecodeMs, r.ResizeMs, r.EncodeMs, r.TotalMs, r.PeakRSSMB, strings.Join(kb, " / "), strings.Join(ss, " / "))
	}
}

// fixturesCmd derives a WebP route-preview stand-in like the map snapshot the
// trail editor stores as the only photo of photo-less trails: 2294x1102, q30.
func fixturesCmd() {
	f, err := os.Open("fixtures/12mp.jpg")
	must(err)
	defer f.Close()
	img, _, err := image.Decode(f)
	must(err)
	crop := imaging.Fill(img, 2294, 1102, imaging.Center, imaging.CatmullRom)
	var buf bytes.Buffer
	must(webp.Encode(&buf, crop, webp.Options{Quality: 30, Method: 4}))
	must(os.WriteFile("fixtures/route.webp", buf.Bytes(), 0o644))
	fmt.Println("fixtures/route.webp", buf.Len(), "bytes")
}

func parseWidths(s string) []int {
	var ws []int
	for _, p := range strings.Split(s, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		must(err)
		ws = append(ws, n)
	}
	return ws
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
