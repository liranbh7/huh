// Package ip resolves an IP address (IPv4 or IPv6) to its hostname, kind, and
// local interface assignment. It uses only the Go standard library.
package ip

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Result holds information about an IP address.
type Result struct {
	Summary   string
	IP        string
	Version   int        // 4 or 6
	Kind      string     // "loopback", "private", "link-local", "multicast", "public", "unspecified"
	Hostname  string     // first result from reverse DNS, empty if none
	Interface string     // local interface name if this IP is assigned to one
	Listeners []Listener // TCP sockets in LISTEN state bound to this IP
	Routes    []Route    // kernel routes that match this IP (IPv4 only)
}

// Listener is a single TCP port in LISTEN state bound to the IP.
type Listener struct {
	Port    int
	PID     int
	Process string
}

// Route is a kernel routing table entry that covers the IP address.
type Route struct {
	Interface string
	Network   string // CIDR notation, e.g. "192.168.1.0/24"
	Gateway   string // empty if directly connected
	Metric    int
}

// Resolve inspects the given IP address string and returns metadata about it.
func Resolve(input string) (*Result, error) {
	parsed := net.ParseIP(input)
	if parsed == nil {
		return nil, fmt.Errorf("ip: invalid address %q", input)
	}

	r := &Result{IP: input}

	if parsed.To4() != nil {
		r.Version = 4
	} else {
		r.Version = 6
	}

	r.Kind = classify(parsed)
	r.Hostname = reverseLookup(parsed)
	r.Interface = localInterface(parsed)
	if r.Interface != "" {
		r.Listeners = findListeners(parsed)
	} else {
		// Routes are only meaningful for addresses not assigned to a local interface.
		r.Routes = findRoutes(parsed)
	}

	if r.Hostname != "" {
		r.Summary = fmt.Sprintf("%s — %s (%s)", input, r.Hostname, r.Kind)
	} else {
		r.Summary = fmt.Sprintf("%s — %s", input, r.Kind)
	}
	return r, nil
}

// classify returns the address category for human display.
func classify(ip net.IP) string {
	switch {
	case ip.IsUnspecified():
		return "unspecified"
	case ip.IsLoopback():
		return "loopback"
	case ip.IsMulticast():
		return "multicast"
	case ip.IsLinkLocalUnicast():
		return "link-local"
	case isPrivate(ip):
		return "private"
	default:
		return "public"
	}
}

// isPrivate reports whether ip falls in RFC 1918 / RFC 4193 private ranges.
func isPrivate(ip net.IP) bool {
	private4 := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	private6 := []string{"fc00::/7"}
	ranges := append(private4, private6...)
	for _, cidr := range ranges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// reverseLookup returns the first PTR record for ip, or empty string.
func reverseLookup(ip net.IP) string {
	names, err := net.LookupAddr(ip.String())
	if err != nil || len(names) == 0 {
		return ""
	}
	// Trim trailing dot from FQDN.
	name := names[0]
	if len(name) > 0 && name[len(name)-1] == '.' {
		name = name[:len(name)-1]
	}
	return name
}

// localInterface returns the name of the local network interface that has ip
// assigned, or empty string if the IP is not local.
func localInterface(ip net.IP) string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ifIP net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ifIP = v.IP
			case *net.IPAddr:
				ifIP = v.IP
			}
			if ifIP != nil && ifIP.Equal(ip) {
				return iface.Name
			}
		}
	}
	return ""
}

// findListeners returns TCP listeners bound to ip or the wildcard address.
func findListeners(ip net.IP) []Listener {
	if ip4 := ip.To4(); ip4 != nil {
		hexIP := fmt.Sprintf("%02X%02X%02X%02X", ip4[3], ip4[2], ip4[1], ip4[0])
		return scanListeners("/proc/net/tcp", hexIP, "00000000")
	}
	hexIP := ipv6Hex(ip)
	return scanListeners("/proc/net/tcp6", hexIP, strings.Repeat("0", 32))
}

// ipv6Hex encodes a 16-byte IPv6 address in the little-endian-per-uint32 hex
// format used by /proc/net/tcp6.
func ipv6Hex(ip net.IP) string {
	b := []byte(ip.To16())
	var sb strings.Builder
	for i := 0; i < 4; i++ {
		g := b[i*4 : i*4+4]
		fmt.Fprintf(&sb, "%02X%02X%02X%02X", g[3], g[2], g[1], g[0])
	}
	return sb.String()
}

// scanListeners reads a /proc/net/tcp[6] file and returns all LISTEN sockets
// whose local address matches hexIP or the wildcard.
func scanListeners(path, hexIP, wildcard string) []Listener {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var listeners []Listener
	scanner := bufio.NewScanner(f)
	scanner.Scan() // skip header
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 {
			continue
		}
		// fields[3] is the state; 0A = TCP_LISTEN
		if !strings.EqualFold(fields[3], "0A") {
			continue
		}
		localAddr := fields[1]
		colon := strings.LastIndex(localAddr, ":")
		if colon < 0 {
			continue
		}
		addrHex := localAddr[:colon]
		if !strings.EqualFold(addrHex, hexIP) && !strings.EqualFold(addrHex, wildcard) {
			continue
		}
		port, err := strconv.ParseUint(localAddr[colon+1:], 16, 16)
		if err != nil {
			continue
		}
		inode, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			continue
		}
		pid, err := inodeToPID(inode)
		if err != nil {
			continue
		}
		listeners = append(listeners, Listener{
			Port:    int(port),
			PID:     pid,
			Process: readComm(pid),
		})
	}
	return listeners
}

// findRoutes returns kernel routing table entries from /proc/net/route that
// cover ip (IPv4 only; IPv6 routing table has a different format).
func findRoutes(ip net.IP) []Route {
	ip4 := ip.To4()
	if ip4 == nil {
		return nil
	}
	ipUint := uint32(ip4[0])<<24 | uint32(ip4[1])<<16 | uint32(ip4[2])<<8 | uint32(ip4[3])

	f, err := os.Open("/proc/net/route")
	if err != nil {
		return nil
	}
	defer f.Close()

	var routes []Route
	scanner := bufio.NewScanner(f)
	scanner.Scan() // skip header
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 8 {
			continue
		}
		dest, err := hexToIP(fields[1])
		if err != nil {
			continue
		}
		gw, err := hexToIP(fields[2])
		if err != nil {
			continue
		}
		mask, err := hexToIP(fields[7])
		if err != nil {
			continue
		}
		metric, _ := strconv.Atoi(fields[6])

		destUint := uint32(dest[0])<<24 | uint32(dest[1])<<16 | uint32(dest[2])<<8 | uint32(dest[3])
		maskUint := uint32(mask[0])<<24 | uint32(mask[1])<<16 | uint32(mask[2])<<8 | uint32(mask[3])
		if ipUint&maskUint != destUint {
			continue
		}

		network := (&net.IPNet{IP: net.IP(dest), Mask: net.IPMask(mask)}).String()
		gwStr := ""
		if gw[0] != 0 || gw[1] != 0 || gw[2] != 0 || gw[3] != 0 {
			gwStr = net.IP(gw).String()
		}
		routes = append(routes, Route{
			Interface: fields[0],
			Network:   network,
			Gateway:   gwStr,
			Metric:    metric,
		})
	}
	return routes
}

// hexToIP parses a little-endian 32-bit hex string (as used in /proc/net/route)
// into a net.IP.
func hexToIP(s string) (net.IP, error) {
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return nil, err
	}
	v := uint32(n)
	return net.IP{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}, nil
}

func inodeToPID(inode uint64) (int, error) {
	target := fmt.Sprintf("socket:[%d]", inode)
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, err
	}
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		fdDir := fmt.Sprintf("/proc/%d/fd", pid)
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}
		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err == nil && link == target {
				return pid, nil
			}
		}
	}
	return 0, fmt.Errorf("no process owns inode %d", inode)
}

func readComm(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(data))
}
