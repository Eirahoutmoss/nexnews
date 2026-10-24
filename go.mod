module nexnews

go 1.24.0

toolchain go1.24.7

require (
	github.com/jchv/go-webview2 v0.0.0-20260205173254-56598839c808
	golang.org/x/net v0.39.0
	golang.org/x/sys v0.39.0
)

require (
	github.com/jchv/go-winloader v0.0.0-20250406163304-c1995be93bd1 // indirect
	golang.org/x/text v0.24.0 // indirect
)

replace golang.org/x/net => github.com/golang/net v0.39.0

replace golang.org/x/sys => github.com/golang/sys v0.39.0

replace golang.org/x/text => github.com/golang/text v0.24.0

replace golang.org/x/crypto => github.com/golang/crypto v0.37.0

replace golang.org/x/term => github.com/golang/term v0.31.0

// NexNews: pencere WS_EX_NOREDIRECTIONBITMAP ile oluşturulur (saydam widget).
replace github.com/jchv/go-webview2 => ./third_party/go-webview2
