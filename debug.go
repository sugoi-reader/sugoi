package main

import (
	"fmt"
)

func debugPrintf(format string, a ...interface{}) {
	if config.Debug {
		fmt.Printf(format, a...)
		return
	}
}
