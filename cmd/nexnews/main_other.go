//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "NexNews masaüstü sürümü yalnızca Windows'ta çalışır (WebView2). Android için android/ klasörüne bakın.")
	os.Exit(1)
}
