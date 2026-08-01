package arkose

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
)

// DeviceIdentity is the FULL per-solve device-spoofer bundle. Every value here would, on a
// real browser, be a hash / count / measurement derived from the specific machine's install
// (GPU driver revision, extension list, math routine implementations, browser build metadata,
// etc.). Previously each of these was a HARDCODED CONSTANT — meaning every request from every
// "device profile" this library shipped emitted the SAME hash. That is a fleet-level tell:
// a classifier just needs to see the same 32-char hex twice from two different "machines" to
// know it's the same solver, regardless of GPU/screen rotation.
//
// The fix: per solve, roll a fresh 32-hex-char salt and derive every "device-specific" hash
// from it (md5(salt|field)). All fields are coherent within one request (same salt), and every
// solve gets a new salt, so two consecutive requests look like two entirely separate machines.
//
// Fields that are TRULY UNIVERSAL for a Chrome build (audio_codecs support map, browser
// constants map, "f" = MD5(""), fe = enumeration of standard flags) are NOT randomized —
// randomizing them would break coherence with the reported UA. Only fields that a real browser
// derives from PER-MACHINE state get rotated.
type DeviceIdentity struct {
	Salt string // 32-hex per-solve salt; every hash below is md5(salt|"<field>")

	// wh second-half — window hash tail. Real Chrome captures showed different values across
	// different real machines. Was hardcoded to a single value for every request.
	MachineHash string

	// Obfuscated 32-hex hash fields in enhanced_fp. Each on a real browser is a hash of some
	// subsystem's state (math routines, browser object surface, TTS voices, speech engines,
	// gamepad list, etc.). All were hardcoded before.
	Hash7541c2s        string
	MathFingerprint    string
	SupportedMathFuncs string
	Hash6a62b2a558     string
	Hashc2d2015        string
	Hash05d3d24        string
	Hash83eb055        string
	Hash1f220c9        string

	// Per-request network telemetry. Real Chrome varies these based on the actual connection.
	NavConnectionDownlink     float64     // real often 1.5–10 depending on link
	NavConnectionDownlink_Max interface{} // real is null or a Mbps number
	NetworkInfoRTT            int         // real 50–300 ms
	ScreenPixelDepth          int         // real 24 on Windows (was fixed 96 = bug)

	// Real Chrome's window tree_structure varies by page content and iframes.
	TreeStructure string
}

// randHexBytes returns n bytes of hex-encoded randomness.
func randHexBytes(n int) string {
	b := make([]byte, n)
	rand.Read(b) //nolint:errcheck // math/rand.Read is always nil-err
	return hex.EncodeToString(b)
}

// hashField derives a deterministic per-salt hex hash for the given field name.
// Same salt+field → same hash (coherence within one solve).
func hashField(salt, field string) string {
	h := md5.Sum([]byte(salt + "|" + field))
	return hex.EncodeToString(h[:])
}

// NewSession rolls a fresh DeviceIdentity for one solve. Call once per Solver.Solve() so all
// BDA hash fields in that request derive from the same salt and look like a coherent machine.
func NewSession() *DeviceIdentity {
	salt := randHexBytes(16)

	// Real Chrome-Windows telemetry distributions (captured 2026-08-01):
	//   downlink: {1.5, 2.5, 5, 10} — 10 is the capped/synthetic default
	//   rtt:      {50, 100, 150, 200, 250, 300}
	//   pixel_depth: Windows Chrome reports 24
	downlinkPool := []float64{1.5, 2.5, 5, 10, 10, 10}
	rttPool := []int{50, 100, 150, 200, 250, 300}

	treeStructures := []string{
		"[[[],[]],[[]],[],[]]",
		"[[],[],[],[[]],[]]",
		"[[],[],[],[[]],[[]],[],[]]",
	}

	return &DeviceIdentity{
		Salt:                      salt,
		MachineHash:               hashField(salt, "machine_hash"),
		Hash7541c2s:               hashField(salt, "7541c2s"),
		MathFingerprint:           hashField(salt, "math_fingerprint"),
		SupportedMathFuncs:        hashField(salt, "supported_math_functions"),
		Hash6a62b2a558:            hashField(salt, "6a62b2a558"),
		Hashc2d2015:               hashField(salt, "c2d2015"),
		Hash05d3d24:               hashField(salt, "05d3d24"),
		Hash83eb055:               hashField(salt, "83eb055"),
		Hash1f220c9:               hashField(salt, "1f220c9"),
		NavConnectionDownlink:     downlinkPool[rand.Intn(len(downlinkPool))],
		NavConnectionDownlink_Max: nil,
		NetworkInfoRTT:            rttPool[rand.Intn(len(rttPool))],
		ScreenPixelDepth:          24,
		TreeStructure:             treeStructures[rand.Intn(len(treeStructures))],
	}
}

// Pick1f220c9 returns either the deterministic hash for this session or null (50/50), matching
// the real Chrome pattern where this field alternates between present and absent.
func (d *DeviceIdentity) Pick1f220c9() interface{} {
	if rand.Intn(2) == 0 {
		return nil
	}
	return d.Hash1f220c9
}

// String is a compact tag for logging. Includes the salt short form so log lines can be
// correlated back to which "device" produced which token.
func (d *DeviceIdentity) String() string {
	return fmt.Sprintf("device[salt=%s…]", d.Salt[:8])
}
