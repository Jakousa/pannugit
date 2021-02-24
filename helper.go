package main

import (
	"fmt"
	"os"
)

// CheckIfError is used for super naive error handling during prototyping
func CheckIfError(err error) {
	if err == nil {
		return
	}

	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", fmt.Sprintf("error: %s", err))
	os.Exit(1)
}