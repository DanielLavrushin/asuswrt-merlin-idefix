// tz.go
package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"os"
	"strings"
	"time"
)

// The router describes its timezone as a POSIX TZ string — /etc/TZ holds
// something like "EST5EDT,M3.2.0/2,M11.1.0/2", not an IANA name such as
// "America/New_York". time.LoadLocation only understands names, so handing it
// /etc/TZ verbatim fails and time.Local silently stays UTC. That is what put
// IDEFIX[pid] lines four hours ahead of every other line in the system log:
// log/syslog stamps each message itself, in local time, before handing it to
// syslogd, which takes the stamp at face value.
//
// tzDataFromPOSIX wraps the POSIX string in a minimal TZif v2 blob — no
// transitions, one placeholder zone, the string in the footer — so that
// LoadLocationFromTZData hands it to the same tzset() the standard library uses
// for the trailing rule of a real zoneinfo file. DST rules then keep being
// evaluated for as long as the daemon runs, instead of being frozen at startup.
//
// placeholderZone is the abbreviation of that one zone, and doubles as a
// tripwire: a Location that still reports it has fallen back on the placeholder
// instead of the footer, which means the POSIX string was not usable.
const placeholderZone = "\x01\x01\x01"

func tzDataFromPOSIX(tz string) []byte {
	abbrev := append([]byte(placeholderZone), 0)

	// Header and data block are written twice: the 32-bit form (which readers
	// skip when the version is 2) and the 64-bit form the footer belongs to.
	header := func() []byte {
		b := append([]byte("TZif2"), make([]byte, 15)...)
		// isutcnt, isstdcnt, leapcnt, timecnt, typecnt, charcnt
		for _, n := range []uint32{0, 0, 0, 0, 1, uint32(len(abbrev))} {
			b = binary.BigEndian.AppendUint32(b, n)
		}
		return b
	}
	block := func() []byte {
		b := binary.BigEndian.AppendUint32(nil, 0) // utoff 0
		b = append(b, 0, 0)                        // isdst 0, abbrev index 0
		return append(b, abbrev...)
	}

	out := append(header(), block()...)
	out = append(out, header()...)
	out = append(out, block()...)
	out = append(out, '\n')
	out = append(out, tz...)
	return append(out, '\n')
}

// loadPOSIXTZ turns a POSIX TZ string into a Location, or reports false if the
// string is not one (an IANA name, say, or an empty /etc/TZ).
func loadPOSIXTZ(tz string) (*time.Location, bool) {
	// A '/' before the first rule means an IANA name or a path rather than a
	// POSIX string (inside a rule it only separates the time of day), and a
	// newline anywhere would break the footer.
	head, _, _ := strings.Cut(tz, ",")
	if tz == "" || strings.Contains(head, "/") || strings.Contains(tz, "\n") {
		return nil, false
	}
	loc, err := time.LoadLocationFromTZData("Local", tzDataFromPOSIX(tz))
	if err != nil {
		return nil, false
	}
	// A malformed footer is not an error: the loader keeps the placeholder
	// zone and the Location quietly means UTC. Probe both halves of the year
	// so a broken DST rule is caught as well as a broken offset.
	for _, month := range []time.Month{time.January, time.July} {
		name, _ := time.Date(time.Now().Year(), month, 15, 12, 0, 0, 0, time.UTC).In(loc).Zone()
		if name == placeholderZone {
			return nil, false
		}
	}
	return loc, true
}

// tzSetting returns the timezone the router is configured for, preferring an
// explicit TZ (control.sh exports one) over /etc/TZ.
func tzSetting() string {
	if tz := strings.TrimSpace(os.Getenv("TZ")); tz != "" {
		return tz
	}
	if raw, err := os.ReadFile("/etc/TZ"); err == nil {
		return string(bytes.TrimSpace(raw))
	}
	return ""
}

// localLocation resolves the local timezone, returning the source it came from
// for logging. A nil Location means nothing could be resolved.
func localLocation(tz string) (*time.Location, string) {
	// POSIX allows a leading ':', and what follows is then a name or, if it
	// starts with '/', the path of a zoneinfo file.
	name := strings.TrimPrefix(tz, ":")

	if loc, ok := loadPOSIXTZ(name); ok {
		return loc, "TZ=" + name
	}
	if name != "" && !strings.HasPrefix(name, "/") {
		if loc, err := time.LoadLocation(name); err == nil {
			return loc, "TZ=" + name
		}
	}

	// Fall back on zoneinfo files: the one TZ points at, then the system's.
	paths := []string{"/etc/localtime"}
	if strings.HasPrefix(name, "/") {
		paths = append([]string{name}, paths...)
	}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if loc, err := time.LoadLocationFromTZData("Local", raw); err == nil {
			return loc, path
		}
	}
	return nil, ""
}

func initTZ() {
	tz := tzSetting()
	if tz != "" {
		// Shells started from the terminal inherit this, so `date` in a
		// session agrees with the router even when TZ was only in /etc/TZ.
		os.Setenv("TZ", tz)
	}

	loc, source := localLocation(tz)
	if loc == nil {
		log.Printf("could not determine the local timezone (TZ=%q); log timestamps will be UTC", tz)
		return
	}
	time.Local = loc
	log.Printf("local time is %s (from %s)", time.Now().Format("2006-01-02 15:04:05 -07:00"), source)
}
