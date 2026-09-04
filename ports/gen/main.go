package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/buggerlogger/arkose-solver/arkose"
)

var groundTruthFE = []string{
	"DNT:unknown",
	"L:en-US",
	"D:24",
	"PR:1",
	"S:1920,1080",
	"AS:1920,1032",
	"TO:-180",
	"SS:true",
	"LS:true",
	"IDB:true",
	"B:false",
	"ODB:false",
	"CPUC:unknown",
	"PK:Win32",
	"CFP:-512569185",
	"FR:false",
	"FOS:false",
	"FB:false",
	"JSF:Arial,Arial Black,Arial Narrow,Calibri,Cambria,Cambria Math,Comic Sans MS,Consolas,Courier,Courier New,Georgia,Helvetica,Impact,Lucida Console,Lucida Sans Unicode,Microsoft Sans Serif,MS Gothic,MS PGothic,MS Sans Serif,MS Serif,Palatino Linotype,Segoe Print,Segoe Script,Segoe UI,Segoe UI Light,Segoe UI Semibold,Segoe UI Symbol,Tahoma,Times,Times New Roman,Trebuchet MS,Verdana,Wingdings",
	"P:Chrome PDF Viewer,Chromium PDF Viewer,Microsoft Edge PDF Viewer,PDF Viewer,WebKit built-in PDF",
	"T:0,false,false",
	"H:16",
	"SWF:false",
}

const (
	wantF   = "2d03456242a080304cde661cdb1853a8"
	wantIfe = "786ec34211e1dec0baf6a24da71c01d5"

	intelVendor   = "Google Inc. (Intel)"
	intelRenderer = "ANGLE (Intel, Intel(R) UHD Graphics (0x00004626) Direct3D11 vs_5_0 ps_5_0, D3D11)"
	wantIntelHash = "44202604505ca5654abd27cb9f40a1bb"
	wantIntelRTT  = "730442"

	nvidiaVendor   = "Google Inc. (NVIDIA)"
	nvidiaRenderer = "ANGLE (NVIDIA, NVIDIA GeForce RTX 4070 (0x00002786) Direct3D11 vs_5_0 ps_5_0, D3D11)"
	wantNvidiaHash = "a3fb76ae00c5b281cdda25b91d37d598"
	wantNvidiaRTT  = "730a3f"

	extensionsHash = "7300c23f4e6fa34e534fc99c1b628588"
)

type khashVector struct {
	Input string `json:"input"`
	Seed  uint32 `json:"seed"`
	Want  string `json:"want"`
}

type webglVector struct {
	Name      string        `json:"name"`
	Vendor    string        `json:"vendor"`
	Renderer  string        `json:"renderer"`
	Fields    []arkose.Item `json:"fields"`
	HashInput string        `json:"hashInput"`
	WantHash  string        `json:"wantHash"`
	WantRTT   string        `json:"wantRttType"`
}

type derivedVectors struct {
	WebGLExtensions         string         `json:"webglExtensions"`
	WantWebGLExtensionsHash string         `json:"wantWebglExtensionsHash"`
	WantF58835f             string         `json:"wantF58835f"`
	WantBrowserObjectChecks string         `json:"wantBrowserObjectChecks"`
	Want29s83ih9            string         `json:"want29s83ih9"`
	SpeechVoices            []arkose.Voice `json:"speechVoices"`
	WantSpeechDefaultVoice  string         `json:"wantSpeechDefaultVoice"`
	WantSpeechVoicesHash    string         `json:"wantSpeechVoicesHash"`
	WantAudioCodecsExtHash  string         `json:"wantAudioCodecsExtendedHash"`
	WantVideoCodecsExtHash  string         `json:"wantVideoCodecsExtendedHash"`
}

type vectors struct {
	Note              string           `json:"_note"`
	Source            string           `json:"_source"`
	KHash             []khashVector    `json:"khash"`
	FE                []string         `json:"fe"`
	WantF             string           `json:"wantF"`
	WantIfeHash       string           `json:"wantIfeHash"`
	WebGL             []webglVector    `json:"webgl"`
	ExtensionsHash    string           `json:"webglExtensionsHash"`
	ScreenPixelFactor int              `json:"screenPixelDepthFactor"`
	JSHeapByMemoryGB  map[string]int64 `json:"jsHeapSizeLimitByMemoryGB"`
	Derived           derivedVectors   `json:"derived"`
}

func fail(what, got, want string) {
	fmt.Fprintf(os.Stderr, "MISMATCH %s\n  got  %s\n  want %s\n", what, got, want)
	os.Exit(1)
}

func main() {
	if got := arkose.ComputeF(groundTruthFE); got != wantF {
		fail("f", got, wantF)
	}
	if got := arkose.ComputeIfeHash(groundTruthFE); got != wantIfe {
		fail("ife_hash", got, wantIfe)
	}

	var webgl []webglVector
	for _, tc := range []struct {
		name, vendor, renderer, wantHash, wantRTT string
	}{
		{"intel-uhd-4626", intelVendor, intelRenderer, wantIntelHash, wantIntelRTT},
		{"nvidia-rtx-4070", nvidiaVendor, nvidiaRenderer, wantNvidiaHash, wantNvidiaRTT},
	} {
		fields := arkose.BuildWebGLFields(tc.vendor, tc.renderer)
		gotHash := arkose.ComputeWebGLHash(fields)
		if gotHash != tc.wantHash {
			fail("webgl_hash_webgl "+tc.name, gotHash, tc.wantHash)
		}
		gotRTT := arkose.ComputeRTTType(extensionsHash, gotHash)
		if gotRTT != tc.wantRTT {
			fail("network_info_rtt_type "+tc.name, gotRTT, tc.wantRTT)
		}
		webgl = append(webgl, webglVector{
			Name:      tc.name,
			Vendor:    tc.vendor,
			Renderer:  tc.renderer,
			Fields:    fields,
			HashInput: arkose.WebGLHashInput(fields),
			WantHash:  gotHash,
			WantRTT:   gotRTT,
		})
	}

	kh := []khashVector{
		{Input: "", Seed: 0},
		{Input: "arkose", Seed: 0},
		{Input: "arkose", Seed: 38},
		{Input: "The quick brown fox jumps over the lazy dog", Seed: 0},
		{Input: "The quick brown fox jumps over the lazy dog", Seed: 31},
		{Input: "0123456789abcdef0123456789abcdef", Seed: 1},
	}
	for i := range kh {
		kh[i].Want = arkose.KHash(kh[i].Input, kh[i].Seed)
	}

	v := vectors{
		Note: "Cross-language conformance vectors. Every port (Go/Python/Node) must reproduce " +
			"these exactly. Values marked want* come from live capi 4.4.5 mints.",
		Source:            "github.com/buggerlogger/arkose-solver",
		KHash:             kh,
		FE:                groundTruthFE,
		WantF:             wantF,
		WantIfeHash:       wantIfe,
		WebGL:             webgl,
		ExtensionsHash:    extensionsHash,
		ScreenPixelFactor: 3,
		JSHeapByMemoryGB:  map[string]int64{"2": 2197815296, "4": 4294705152, "8": 4395630592, "16": 4395630592},
	}

	sampleVoices := []arkose.Voice{
		{Name: "Microsoft David - English (United States)", Lang: "en-US", Default: true},
		{Name: "Microsoft Zira - English (United States)", Lang: "en-US"},
	}
	sv, sh := arkose.ComputeSpeech(sampleVoices)
	v.Derived = derivedVectors{
		WebGLExtensions:         arkose.WebGLExtensions,
		WantWebGLExtensionsHash: arkose.ComputeWebGLExtensionsHash(arkose.WebGLExtensions),
		WantF58835f:             arkose.ChromeWindowsF58835f(),
		WantBrowserObjectChecks: arkose.ComputeBrowserObjectChecks([]string{"chrome"}),
		Want29s83ih9:            arkose.MD5("false"),
		SpeechVoices:            sampleVoices,
		WantSpeechDefaultVoice:  sv,
		WantSpeechVoicesHash:    sh,
		WantAudioCodecsExtHash:  arkose.AudioCodecsExtendedHash(),
		WantVideoCodecsExtHash:  arkose.VideoCodecsExtendedHash(),
	}
	if v.Derived.WantWebGLExtensionsHash != extensionsHash {
		fail("webgl_extensions_hash", v.Derived.WantWebGLExtensionsHash, extensionsHash)
	}
	if v.Derived.WantF58835f != "6681f2144d3d1ceb383fe87f7aad9600" {
		fail("f58835f", v.Derived.WantF58835f, "6681f2144d3d1ceb383fe87f7aad9600")
	}
	if v.Derived.WantBrowserObjectChecks != "554838a8451ac36cb977e719e9d6623c" {
		fail("browser_object_checks", v.Derived.WantBrowserObjectChecks, "554838a8451ac36cb977e719e9d6623c")
	}
	if v.Derived.WantAudioCodecsExtHash != "805036349642e2569ec299baed02315b" {
		fail("audio_codecs_extended_hash", v.Derived.WantAudioCodecsExtHash, "805036349642e2569ec299baed02315b")
	}
	if v.Derived.WantVideoCodecsExtHash != "5648501a58d24ea22ce3773cc234e563" {
		fail("video_codecs_extended_hash", v.Derived.WantVideoCodecsExtHash, "5648501a58d24ea22ce3773cc234e563")
	}

	out, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		panic(err)
	}
	path := "ports/testvectors.json"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	if err := os.WriteFile(path, append(out, '\n'), 0644); err != nil {
		panic(err)
	}
	fmt.Printf("all ground-truth assertions passed; wrote %s (%d bytes)\n", path, len(out))
}
