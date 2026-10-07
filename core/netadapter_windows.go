//go:build windows

package core

// Live link information for the adapter that carries the default route:
// friendly name, negotiated link speed and the whole computer's current
// download/upload rate (from the interface octet counters, sampled each second).

import (
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type adapterInfo struct {
	ok             bool
	name           string
	linkMbps       int64
	downBps, upBps int64
}

var adapterState struct {
	sync.Mutex
	info adapterInfo
}

var adapterSamplerOnce sync.Once

func currentAdapter() adapterInfo {
	adapterState.Lock()
	defer adapterState.Unlock()
	return adapterState.info
}

type primaryAdapter struct {
	luid     uint64
	name     string
	linkMbps int64
}

// findPrimaryAdapter returns the up, non-loopback adapter that has a gateway
// and the lowest IPv4 metric (i.e. the one Windows routes internet traffic to).
func findPrimaryAdapter() (primaryAdapter, bool) {
	var best primaryAdapter
	found := false
	size := uint32(16 * 1024)
	for i := 0; i < 3; i++ {
		buf := make([]byte, size)
		aa := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
		const flags = windows.GAA_FLAG_INCLUDE_GATEWAYS | windows.GAA_FLAG_SKIP_ANYCAST |
			windows.GAA_FLAG_SKIP_MULTICAST | windows.GAA_FLAG_SKIP_DNS_SERVER
		err := windows.GetAdaptersAddresses(windows.AF_UNSPEC, flags, 0, aa, &size)
		if err == windows.ERROR_BUFFER_OVERFLOW {
			continue
		}
		if err != nil {
			return best, false
		}
		bestMetric := uint32(1<<32 - 1)
		for a := aa; a != nil; a = a.Next {
			if a.OperStatus != windows.IfOperStatusUp || a.IfType == windows.IF_TYPE_SOFTWARE_LOOPBACK || a.FirstGatewayAddress == nil {
				continue
			}
			if a.Ipv4Metric < bestMetric || !found {
				bestMetric = a.Ipv4Metric
				speed := a.ReceiveLinkSpeed
				if speed == 0 || speed == ^uint64(0) {
					speed = a.TransmitLinkSpeed
				}
				if speed == ^uint64(0) {
					speed = 0
				}
				best = primaryAdapter{luid: a.Luid, name: windows.UTF16PtrToString(a.FriendlyName), linkMbps: int64(speed / 1_000_000)}
				found = true
			}
		}
		return best, found
	}
	return best, false
}

func startAdapterSampler() {
	adapterSamplerOnce.Do(func() {
		go func() {
			defer func() { _ = recover() }()
			var (
				pa              primaryAdapter
				have            bool
				lastIn, lastOut uint64
				lastAt          time.Time
				lastLookup      time.Time
			)
			t := time.NewTicker(time.Second)
			defer t.Stop()
			for ; ; <-t.C {
				if !have || time.Since(lastLookup) > 15*time.Second {
					npa, ok := findPrimaryAdapter()
					lastLookup = time.Now()
					if !ok {
						have = false
						adapterState.Lock()
						adapterState.info = adapterInfo{}
						adapterState.Unlock()
						continue
					}
					if !have || npa.luid != pa.luid {
						lastAt = time.Time{}
					}
					pa, have = npa, true
				}
				row := windows.MibIfRow2{InterfaceLuid: pa.luid}
				if err := windows.GetIfEntry2Ex(windows.MibIfEntryNormal, &row); err != nil {
					have = false
					continue
				}
				now := time.Now()
				info := adapterInfo{ok: true, name: pa.name, linkMbps: pa.linkMbps, downBps: -1, upBps: -1}
				if !lastAt.IsZero() && row.InOctets >= lastIn && row.OutOctets >= lastOut {
					dt := now.Sub(lastAt).Seconds()
					if dt > 0 {
						info.downBps = int64(float64(row.InOctets-lastIn) / dt)
						info.upBps = int64(float64(row.OutOctets-lastOut) / dt)
					}
				}
				lastIn, lastOut, lastAt = row.InOctets, row.OutOctets, now
				adapterState.Lock()
				adapterState.info = info
				adapterState.Unlock()
			}
		}()
	})
}
