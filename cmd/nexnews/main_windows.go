//go:build windows

// Command nexnews is the Windows desktop build of NexNews. All logic lives in
// the shared core package (also used by the Android app through gomobile).
package main

import "nexnews/core"

func main() { core.RunDesktop() }
