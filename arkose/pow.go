package arkose

import "errors"

type PoWChallenge map[string]interface{}

func SolvePoW(_ PoWChallenge) (map[string]interface{}, error) {
	return nil, errors.New("PoW challenge received but not implemented (this solver targets sup=1 tokens)")
}
