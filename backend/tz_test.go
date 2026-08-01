package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadPOSIXTZ(t *testing.T) {
	// 2026-04-13T23:22:10Z is the instant from the report that came out of
	// syslog as "Apr 13 23:22:10" instead of the router's 19:22:10.
	instant := time.Date(2026, 4, 13, 23, 22, 10, 0, time.UTC)

	cases := []struct {
		tz     string
		local  string // instant rendered the way log/syslog stamps a message
		winter string // offset in January, to catch a mishandled DST rule
		summer string // ... and in July
	}{
		{"EST5EDT,M3.2.0/2,M11.1.0/2", "Apr 13 19:22:10", "-0500", "-0400"},
		{"EST5EDT", "Apr 13 19:22:10", "-0500", "-0400"}, // rules omitted: US defaults
		{"CET-1CEST,M3.5.0,M10.5.0/3", "Apr 14 01:22:10", "+0100", "+0200"},
		{"AEST-10AEDT,M10.1.0,M4.1.0/3", "Apr 14 09:22:10", "+1100", "+1000"}, // southern hemisphere
		{"MSK-3", "Apr 14 02:22:10", "+0300", "+0300"},                        // no DST
		{"<+0545>-5:45", "Apr 14 05:07:10", "+0545", "+0545"},                 // quoted name, non-hour offset
		{"UTC0", "Apr 13 23:22:10", "+0000", "+0000"},
	}

	for _, c := range cases {
		t.Run(c.tz, func(t *testing.T) {
			loc, ok := loadPOSIXTZ(c.tz)
			if !ok {
				t.Fatalf("loadPOSIXTZ(%q) rejected a valid POSIX TZ string", c.tz)
			}
			if got := instant.In(loc).Format(time.Stamp); got != c.local {
				t.Errorf("stamp = %q, want %q", got, c.local)
			}
			winter := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC).In(loc).Format("-0700")
			if winter != c.winter {
				t.Errorf("January offset = %s, want %s", winter, c.winter)
			}
			summer := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC).In(loc).Format("-0700")
			if summer != c.summer {
				t.Errorf("July offset = %s, want %s", summer, c.summer)
			}
		})
	}
}

func TestLoadPOSIXTZRejectsNonPOSIX(t *testing.T) {
	// Anything we cannot evaluate has to be rejected rather than quietly
	// yielding a Location that means UTC — that is the bug being fixed.
	for _, tz := range []string{
		"",
		"garbage",
		"America/New_York",         // IANA name, handled elsewhere
		"/etc/localtime",           // path, handled elsewhere
		"EST5EDT,M3.2.0/2\n",       // trailing newline would break the footer
		"EST5EDT,M3.2.0/2",         // start rule without an end rule
		"EST5EDT,J400/2,M11.1.0/2", // day out of range
	} {
		if loc, ok := loadPOSIXTZ(tz); ok {
			t.Errorf("loadPOSIXTZ(%q) = %v, want rejection", tz, loc)
		}
	}
}

func TestLocalLocation(t *testing.T) {
	t.Run("POSIX TZ string", func(t *testing.T) {
		loc, source := localLocation("EST5EDT,M3.2.0/2,M11.1.0/2")
		if loc == nil {
			t.Fatal("no location resolved")
		}
		if _, offset := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC).In(loc).Zone(); offset != -4*60*60 {
			t.Errorf("July offset = %ds, want %ds", offset, -4*60*60)
		}
		if source == "" {
			t.Error("source not reported")
		}
	})

	t.Run("IANA name", func(t *testing.T) {
		loc, _ := localLocation("Europe/Berlin")
		if loc == nil {
			t.Fatal("no location resolved")
		}
		if _, offset := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC).In(loc).Zone(); offset != 2*60*60 {
			t.Errorf("July offset = %ds, want %ds", offset, 2*60*60)
		}
	})

	t.Run("path to a zoneinfo file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "localtime")
		if err := os.WriteFile(path, tzDataFromPOSIX("MSK-3"), 0o644); err != nil {
			t.Fatal(err)
		}
		loc, source := localLocation(":" + path)
		if loc == nil {
			t.Fatal("no location resolved")
		}
		if _, offset := time.Now().In(loc).Zone(); offset != 3*60*60 {
			t.Errorf("offset = %ds, want %ds", offset, 3*60*60)
		}
		if source != path {
			t.Errorf("source = %q, want %q", source, path)
		}
	})

	t.Run("nothing usable", func(t *testing.T) {
		// Only fails on a host without /etc/localtime, so accept either
		// outcome as long as garbage never turns into a bogus zone.
		if loc, _ := localLocation("garbage"); loc != nil {
			if name, _ := time.Now().In(loc).Zone(); name == placeholderZone {
				t.Errorf("garbage TZ resolved to the placeholder zone")
			}
		}
	})
}

func TestInitTZ(t *testing.T) {
	t.Setenv("TZ", "EST5EDT,M3.2.0/2,M11.1.0/2")
	saved := time.Local
	t.Cleanup(func() { time.Local = saved })

	initTZ()

	instant := time.Date(2026, 4, 13, 23, 22, 10, 0, time.UTC)
	if got := instant.In(time.Local).Format(time.Stamp); got != "Apr 13 19:22:10" {
		t.Fatalf("syslog stamp = %q, want %q", got, "Apr 13 19:22:10")
	}
}
