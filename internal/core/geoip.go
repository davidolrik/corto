package core

import (
	"fmt"
	"net"

	"github.com/oschwald/geoip2-golang"
)

// GeoIP resolves IP addresses to ISO 3166-1 country codes using a MaxMind
// format database (GeoLite2-Country, DB-IP country lite, or similar).
type GeoIP struct {
	reader *geoip2.Reader
}

func NewGeoIP(databasePath string) (*GeoIP, error) {
	reader, err := geoip2.Open(databasePath)
	if err != nil {
		return nil, fmt.Errorf("opening GeoIP database %q: %w", databasePath, err)
	}
	return &GeoIP{reader: reader}, nil
}

// Country returns the ISO country code for the given IP address, or an empty
// string when the IP is invalid or not in the database.
func (g *GeoIP) Country(ipAddress string) string {
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return ""
	}
	record, err := g.reader.Country(ip)
	if err != nil {
		return ""
	}
	return record.Country.IsoCode
}

func (g *GeoIP) Close() error {
	return g.reader.Close()
}
