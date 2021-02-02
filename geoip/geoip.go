package geoip

import (
	"fmt"
	"math"
	"net"
	"sort"

	"github.com/DragonFlyBSD/mirrorselect/common"
)

var appConfig = common.AppConfig

type Location struct {
	ContinentCode	string
	CountryCode	string
	Latitude	float64
	Longitude	float64
}

type Point struct {
	Latitude	float64
	Longitude	float64
}

// MaxMind and DB-IP use the same schema for the selected fields here.
type Record struct {
	Continent struct {
		// Two-letter code
		Code		string  `maxminddb:"code"`
	} `maxminddb:"continent"`
	Country struct {
		// Two-letter code
		Code		string  `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	Location struct {
		Latitude	float64 `maxminddb:"latitude"`
		Longitude	float64 `maxminddb:"longitude"`
	} `maxminddb:"location"`
}


// Lookup the location data in MMDB for the IP address.
//
func LookupIP(ip net.IP) (*Location, error) {
	var record Record
	_, ok, err := appConfig.MMDB.DB.LookupNetwork(ip, &record)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("No data for IP (%s)", ip.String())
	}

	location := Location{
		ContinentCode: record.Continent.Code,
		CountryCode:   record.Country.Code,
		Latitude:      record.Location.Latitude,
		Longitude:     record.Location.Longitude,
	}
	return &location, nil
}


// Find mirrors that suit the given location.
//
// Rules:
// - Prefer mirrors of the same country.
// - Then prefer mirrors of the same continent.
// - Fallback to the default mirror.
// - If multiple mirrors in the same country/continent, order by
//   distance via latitude/longitude.
// - Append the default to the last as the fallback.
// - If location is nil, then return the default mirror.
//
func FindMirrors(location *Location) []*common.Mirror {
	if location == nil {
		// Return the default mirror
		for _, mirror := range appConfig.Mirrors {
			if mirror.IsDefault {
				return []*common.Mirror{ mirror }
			}
		}
	}

	var m_default *common.Mirror
	var m_country, m_continent []*common.Mirror
	for _, mirror := range appConfig.Mirrors {
		if mirror.IsDefault {
			// Always use it even if offline
			m_default = mirror
		}
		if !mirror.Status.Online {
			continue
		}
		if mirror.CountryCode == location.CountryCode {
			m_country = append(m_country, mirror)
		}
		if mirror.ContinentCode == location.ContinentCode {
			m_continent = append(m_continent, mirror)
		}
	}

	sort.Slice(m_country, fLess(m_country, location))
	sort.Slice(m_continent, fLess(m_continent, location))

	mirrors := []*common.Mirror{}
	if len(m_country) > 0 {
		mirrors = append(mirrors, m_country...)
	} else if len(m_continent) > 0 {
		mirrors = append(mirrors, m_continent...)
	}
	// Append the default mirror as fallback
	mirrors = append(mirrors, m_default)

	return mirrors
}


// Helper function that returns another function to sort the mirror
// slice by their distances to the client.
//
func fLess(s []*common.Mirror, loc *Location) func(i, j int) bool {
	return func(i, j int) bool {
		di := mirrorDistance(s[i], loc)
		dj := mirrorDistance(s[j], loc)
		return di < dj
	}
}

// Helper function to calculate the distance of mirror to the client.
//
func mirrorDistance(mirror *common.Mirror, loc *Location) float64 {
	point1 := Point{
		Latitude: loc.Latitude,
		Longitude: loc.Longitude,
	}
	point2 := Point{
		Latitude: mirror.Latitude,
		Longitude: mirror.Longitude,
	}
	return greatCircleDistance(&point1, &point2)
}

// Calculate the great-circle distance between two points on Earth.
//
// References:
// - https://en.wikipedia.org/wiki/Great-circle_distance#Computational_formulas
// - http://edwilliams.org/avform.htm
//
func greatCircleDistance(loc1, loc2 *Point) float64 {
	lat1 := loc1.Latitude  * math.Pi / 180.0
	lon1 := loc1.Longitude * math.Pi / 180.0
	lat2 := loc2.Latitude  * math.Pi / 180.0
	lon2 := loc2.Longitude * math.Pi / 180.0

	t1 := math.Pow(math.Sin((lat1-lat2)/2), 2)
	t2 := math.Cos(lat1) * math.Cos(lat2) * math.Pow(math.Sin((lon1-lon2)/2), 2)
	d := 2 * math.Asin(math.Sqrt(t1 + t2))
	return d
}
