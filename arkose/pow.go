package arkose

import "errors"

// PoWChallenge is a placeholder for the /pows/setup response. The Python solver shells
// out to a Node.js sandbox.js (missing from the shipped project) to run the actual PoW
// algorithm, so faithfully porting the compute step is not possible from Python alone.
//
// For iStockPhoto tokens the /fc/gt2/public_key/ response almost always carries `sup=1`
// (bypass), meaning PoW is skipped anyway. If a PoW is ever demanded, we surface a clean
// error rather than silently returning an invalid token.
type PoWChallenge map[string]interface{}

// SolvePoW is a stub — see the note above.
func SolvePoW(_ PoWChallenge) (map[string]interface{}, error) {
	return nil, errors.New("PoW challenge received but Node sandbox is not available in this build")
}
