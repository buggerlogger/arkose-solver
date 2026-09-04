package main

import (
	"fmt"
	"log"

	"github.com/buggerlogger/arkose-solver/arkose"
)

func main() {
	s, err := arkose.New(
		arkose.WithSurl("https://verify.example.com"),
		arkose.WithPublicKey("00000000-0000-0000-0000-000000000000"),
		arkose.WithSite("https://www.example.com"),
		arkose.WithProxy("http://user:pass@host:port"),
	)
	if err != nil {
		log.Fatal(err)
	}

	res, err := s.Solve()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res.Token, "suppressed:", res.Suppressed)
}
