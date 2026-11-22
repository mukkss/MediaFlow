//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("MediaFlow Desktop GUI is built for Windows (Wails v3 + Win32/WebView2).")
	fmt.Println("To run the conversion engine on this environment, use: ./bin/mediaflow -input <path>")
	os.Exit(0)
}
