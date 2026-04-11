// Package sysinfo collects static and dynamic host information for agent registration.
package sysinfo

import (
	"net"
	"os"
	"runtime"
)

// Static holds the information gathered once at startup.
type Static struct {
	Hostname string
	OS       string
	Arch     string
	LocalIPs []string
}

// Gather collects static host information. Errors are soft — partial results
// are returned rather than failing the agent startup.
func Gather() Static {
	hostname, _ := os.Hostname()

	return Static{
		Hostname: hostname,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		LocalIPs: localIPs(),
	}
}

func localIPs() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.To4() != nil {
				ips = append(ips, ip.String())
			}
		}
	}
	return ips
}
