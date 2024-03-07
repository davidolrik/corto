package core_test

import (
	"testing"

	"github.com/davidolrik/corto/internal/core"
)

func TestGeoIPCountry(t *testing.T) {
	geo, err := core.NewGeoIP("testdata/GeoIP2-Country-Test.mmdb")
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	defer geo.Close()

	cases := []struct {
		name string
		ip   string
		want string
	}{
		{name: "known IPv4", ip: "81.2.69.142", want: "GB"},
		{name: "known IPv6", ip: "2001:218::1", want: "JP"},
		{name: "private IP has no country", ip: "192.168.1.1", want: ""},
		{name: "unparsable IP", ip: "not-an-ip", want: ""},
		{name: "empty IP", ip: "", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := geo.Country(tc.ip); got != tc.want {
				t.Errorf("Country(%q) = %q, want %q", tc.ip, got, tc.want)
			}
		})
	}
}

func TestGeoIPMissingDatabase(t *testing.T) {
	_, err := core.NewGeoIP("testdata/does-not-exist.mmdb")
	if err == nil {
		t.Fatal("expected error for missing database file")
	}
}
