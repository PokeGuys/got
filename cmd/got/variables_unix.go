//go:build !windows
// +build !windows

package main

import (
	"fmt"
)

var (
	progressStyle = "block"
	r, l          = "▕", "▏"
)

func color(content ...interface{}) string {
	return fmt.Sprintf("\x1b[34m%s\x1b[0m", fmt.Sprint(content...))
}
