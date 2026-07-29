// Minimal example: solve one Arkose token and print it.
//
//	go run ./examples/basic
package main

import (
	"fmt"
	"log"

	"github.com/yourusername/arkose-solver/arkose"
)

func main() {
	s, err := arkose.New(
		arkose.WithSurl("https://verify.example.com"),         // Arkose verify host
		arkose.WithPublicKey("00000000-0000-0000-0000-000000000000"), // site key (UUID)
		arkose.WithRSAPublicKey("MIIBIjAN...replace-with-your-captured-key..."),
		arkose.WithSite("https://www.example.com"),
		// arkose.WithProxy("http://user:pass@host:port"),
	)
	if err != nil {
		log.Fatal(err)
	}

	res, err := s.Solve()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("token      : %s\n", res.Token)
	fmt.Printf("suppressed : %v (sup=1 == trusted, no challenge)\n", res.Suppressed)
	fmt.Printf("timings    : %v\n", res.Timings)
}
