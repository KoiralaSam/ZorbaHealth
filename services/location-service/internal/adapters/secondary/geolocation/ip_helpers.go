package geolocation

import (
	"net"
	"strings"
)

// IsUngeolocatableIP reports addresses that cannot be resolved by public IP APIs
// (loopback, private ranges, or empty). Common when the browser reaches the pod
// via kubectl port-forward (RemoteAddr appears as 127.0.0.1).
func IsUngeolocatableIP(ip string) bool {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return true
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return true
	}
	return parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsUnspecified()
}
