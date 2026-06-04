package main

import (
	"fmt"
	"os"

	"github.com/niko-admin/niko-admin/internal/pkg/hash"
)

func main() {
	h, err := hash.Hash(os.Args[1])
	if err != nil {
		fmt.Println("ERROR:", err)
		return
	}
	fmt.Println(h)
}
