//go:build windows
// +build windows

package main

import "fmt"

// Windows doesn't handle the block-style very well
var (
	r, l = "[", "]"
)

func color(content ...interface{}) string {
	return fmt.Sprint(content...)
}
