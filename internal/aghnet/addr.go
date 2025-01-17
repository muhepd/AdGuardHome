package aghnet

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/agherr"
	"golang.org/x/net/idna"
)

// IsValidHostOuterRune returns true if r is a valid initial or final rune for
// a hostname label.
func IsValidHostOuterRune(r rune) (ok bool) {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9')
}

// isValidHostRune returns true if r is a valid rune for a hostname label.
func isValidHostRune(r rune) (ok bool) {
	return r == '-' || IsValidHostOuterRune(r)
}

// ValidateHardwareAddress returns an error if hwa is not a valid EUI-48,
// EUI-64, or 20-octet InfiniBand link-layer address.
func ValidateHardwareAddress(hwa net.HardwareAddr) (err error) {
	defer agherr.Annotate("validating hardware address %q: %w", &err, hwa)

	switch l := len(hwa); l {
	case 0:
		return agherr.Error("address is empty")
	case 6, 8, 20:
		return nil
	default:
		return fmt.Errorf("bad len: %d", l)
	}
}

// maxDomainLabelLen is the maximum allowed length of a domain name label
// according to RFC 1035.
const maxDomainLabelLen = 63

// maxDomainNameLen is the maximum allowed length of a full domain name
// according to RFC 1035.
//
// See https://stackoverflow.com/a/32294443/1892060.
const maxDomainNameLen = 253

// ValidateDomainNameLabel returns an error if label is not a valid label of
// a domain name.
func ValidateDomainNameLabel(label string) (err error) {
	defer agherr.Annotate("validating label %q: %w", &err, label)

	l := len(label)
	if l > maxDomainLabelLen {
		return fmt.Errorf("label is too long, max: %d", maxDomainLabelLen)
	} else if l == 0 {
		return agherr.Error("label is empty")
	}

	if r := label[0]; !IsValidHostOuterRune(rune(r)) {
		return fmt.Errorf("invalid char %q at index %d", r, 0)
	} else if l == 1 {
		return nil
	}

	for i, r := range label[1 : l-1] {
		if !isValidHostRune(r) {
			return fmt.Errorf("invalid char %q at index %d", r, i+1)
		}
	}

	if r := label[l-1]; !IsValidHostOuterRune(rune(r)) {
		return fmt.Errorf("invalid char %q at index %d", r, l-1)
	}

	return nil
}

// ValidateDomainName validates the domain name in accordance to RFC 952, RFC
// 1035, and with RFC-1123's inclusion of digits at the start of the host.  It
// doesn't validate against two or more hyphens to allow punycode and
// internationalized domains.
//
// TODO(a.garipov): After making sure that this works correctly, port this into
// module golibs.
func ValidateDomainName(name string) (err error) {
	defer agherr.Annotate("validating domain name %q: %w", &err, name)

	name, err = idna.ToASCII(name)
	if err != nil {
		return err
	}

	l := len(name)
	if l == 0 {
		return agherr.Error("domain name is empty")
	} else if l > maxDomainNameLen {
		return fmt.Errorf("too long, max: %d", maxDomainNameLen)
	}

	labels := strings.Split(name, ".")
	for i, l := range labels {
		err = ValidateDomainNameLabel(l)
		if err != nil {
			return fmt.Errorf("invalid domain name label at index %d: %w", i, err)
		}
	}

	return nil
}

// The maximum lengths of generated hostnames for different IP versions.
const (
	ipv4HostnameMaxLen = len("192-168-100-10-")
	ipv6HostnameMaxLen = len("ff80-f076-0000-0000-0000-0000-0000-0010")
)

// generateIPv4Hostname generates the hostname for specific IP version.
func generateIPv4Hostname(ipv4 net.IP) (hostname string) {
	hnData := make([]byte, 0, ipv4HostnameMaxLen)
	for i, part := range ipv4 {
		if i > 0 {
			hnData = append(hnData, '-')
		}
		hnData = strconv.AppendUint(hnData, uint64(part), 10)
	}

	return string(hnData)
}

// generateIPv6Hostname generates the hostname for specific IP version.
func generateIPv6Hostname(ipv6 net.IP) (hostname string) {
	hnData := make([]byte, 0, ipv6HostnameMaxLen)
	for i, partsNum := 0, net.IPv6len/2; i < partsNum; i++ {
		if i > 0 {
			hnData = append(hnData, '-')
		}
		for _, val := range ipv6[i*2 : i*2+2] {
			if val < 10 {
				hnData = append(hnData, '0')
			}
			hnData = strconv.AppendUint(hnData, uint64(val), 16)
		}
	}

	return string(hnData)
}

// GenerateHostname generates the hostname from ip.  In case of using IPv4 the
// result should be like:
//
//   192-168-10-1
//
// In case of using IPv6, the result is like:
//
//   ff80-f076-0000-0000-0000-0000-0000-0010
//
func GenerateHostname(ip net.IP) (hostname string) {
	if ipv4 := ip.To4(); ipv4 != nil {
		return generateIPv4Hostname(ipv4)
	} else if ipv6 := ip.To16(); ipv6 != nil {
		return generateIPv6Hostname(ipv6)
	}

	return ""
}
