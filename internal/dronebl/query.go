package dronebl

import (
	"encoding/hex"
	"fmt"
	"net/netip"
	"slices"
	"strconv"
	"strings"
)

const zone = "dnsbl.dronebl.org"

func queryName(ip string) (string, bool) {
	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return "", false
	}

	addr = addr.Unmap()
	if addr.IsUnspecified() || addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() {
		return "", false
	}

	if addr.Is4() {
		o := addr.As4()

		return fmt.Sprintf("%d.%d.%d.%d.%s", o[3], o[2], o[1], o[0], zone), true
	}

	octets := addr.As16()

	digits := []byte(hex.EncodeToString(octets[:]))
	slices.Reverse(digits)

	var sb strings.Builder
	for _, digit := range digits {
		sb.WriteByte(digit)
		sb.WriteByte('.')
	}
	sb.WriteString(zone)

	return sb.String(), true
}

func classesFrom(answers []netip.Addr) []int {
	classes := make([]int, 0, len(answers))

	for _, answer := range answers {
		answer = answer.Unmap()
		if !answer.Is4() {
			continue
		}

		if octets := answer.As4(); octets[0] == 127 && octets[3] > 1 {
			classes = append(classes, int(octets[3]))
		}
	}

	return classes
}

func parseClassFilter(raw string) map[int]bool {
	filter := make(map[int]bool)

	for field := range strings.SplitSeq(raw, ",") {
		if class, err := strconv.Atoi(strings.TrimSpace(field)); err == nil && class > 1 && class <= 255 {
			filter[class] = true
		}
	}

	return filter
}

func parseAllowlist(raw string) []netip.Prefix {
	var prefixes []netip.Prefix

	for field := range strings.SplitSeq(raw, ",") {
		field = strings.TrimSpace(field)

		if prefix, err := netip.ParsePrefix(field); err == nil {
			prefixes = append(prefixes, prefix)
			continue
		}

		if addr, err := netip.ParseAddr(field); err == nil {
			prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
		}
	}

	return prefixes
}

func allowlisted(prefixes []netip.Prefix, ip string) bool {
	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return false
	}

	addr = addr.Unmap()

	return slices.ContainsFunc(prefixes, func(prefix netip.Prefix) bool {
		return prefix.Contains(addr)
	})
}
