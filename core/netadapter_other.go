//go:build !windows

package core

type adapterInfo struct {
	ok             bool
	name           string
	linkMbps       int64
	downBps, upBps int64
}

func startAdapterSampler()        {}
func currentAdapter() adapterInfo { return adapterInfo{} }
