package arkose

import "errors"

// PoWChallenge is a placeholder for the /pows/setup response. It is intentionally unused:
// a suppressed (sup=1) token means the server demands no challenge, so no PoW is ever run.
// It is kept only so callers that branch on a non-sup=1 outcome have a typed stub to reference.
type PoWChallenge map[string]interface{}

// SolvePoW is a stub. This solver targets sup=1 tokens (no challenge). If a deployment ever
// demanded a PoW, that would be a separate, unimplemented step.
func SolvePoW(_ PoWChallenge) (map[string]interface{}, error) {
	return nil, errors.New("PoW challenge received but not implemented (this solver targets sup=1 tokens)")
}
