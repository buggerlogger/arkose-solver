package arkose

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestGenDevices(t *testing.T) {
	const n = 12
	seenGPU := map[string]int{}
	seenHash := map[string]bool{}
	for i := 0; i < n; i++ {
		d := GenerateDevice()
		w := d.WebGL()
		iw, ih := d.Inner()
		if seenHash[w.HashWebGL] {
			t.Errorf("duplicate webgl hash across solves: %s", w.HashWebGL)
		}
		seenHash[w.HashWebGL] = true
		seenGPU[d.GPUName]++

		if iw >= d.OuterWidth || ih >= d.OuterHeight {
			t.Errorf("incoherent window: inner(%d,%d) !< outer(%d,%d)", iw, ih, d.OuterWidth, d.OuterHeight)
		}
		if d.DeviceMemory < 4 || d.HardwareConcurrency < 4 {
			t.Errorf("implausible mem/cores: %d/%d", d.DeviceMemory, d.HardwareConcurrency)
		}
		if len(w.HashWebGL) != 32 {
			t.Errorf("webgl hash not 32-hex: %q", w.HashWebGL)
		}
		if os.Getenv("VERBOSE_DEV") != "" {
			b, _ := json.Marshal(map[string]any{
				"gpu": w.UnmaskedRenderer, "hash": w.HashWebGL,
				"screen": fmt.Sprintf("%dx%d", d.OuterWidth, d.OuterHeight+48),
				"inner":  fmt.Sprintf("%dx%d", iw, ih),
				"mem":    d.DeviceMemory, "cores": d.HardwareConcurrency,
			})
			t.Logf("dev %2d %s", i, b)
		}
	}
	t.Logf("distinct webgl hashes: %d/%d, distinct GPUs: %d", len(seenHash), n, len(seenGPU))
}

func TestDumpBDA(t *testing.T) {
	out := os.Getenv("DUMP_OUT")
	if out == "" {
		t.Skip("set DUMP_OUT=<path> (and optionally DUMP_SITE/DUMP_SURL/DUMP_PK/DUMP_TITLE/DUMP_BUILDID) to dump a BDA plaintext")
	}
	site := envOr("DUMP_SITE", "https://www.example.com")
	cfg := &Config{
		Surl:                 envOr("DUMP_SURL", "https://verify.example.com"),
		PublicKey:            envOr("DUMP_PK", "00000000-0000-0000-0000-000000000000"),
		Site:                 site,
		WindowLocationHref:   site + "/sign-in",
		SitedataLocationHref: site + "/sign-in",
		Language:             "en-US",
		Title:                envOr("DUMP_TITLE", "Sign in - Example"),
	}
	cfg.applyDefaults()
	bda := generateBDA(cfg, os.Getenv("DUMP_BUILDID"))
	b, err := marshalBDA(bda)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, b, 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d bytes to %s", len(b), out)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
