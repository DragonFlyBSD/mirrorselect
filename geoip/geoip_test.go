package geoip

import (
	"math"
	"net"
	"testing"

	"github.com/DragonFlyBSD/mirrorselect/common"
)


func init() {
	fname := "../testdata/mirrorselect.toml"
	common.ReadConfig(fname)
}


func TestLookupIP1(t *testing.T) {
	unknown_ips := []string{ "127.0.0.1", "::1" }

	for _, ipstr := range unknown_ips {
		ip := net.ParseIP(ipstr)
		loc, _ := LookupIP(ip)
		if loc != nil {
			t.Errorf("LookupIP(%v) = %v, want %v", ip, loc, nil)
		}
	}
}


func TestLookupIP2(t *testing.T) {
	cases := []struct {
		ip string
		location *Location
	}{
		{
			ip: "199.233.90.68",  // leaf.dragonflybsd.org
			location: &Location{
				ContinentCode: "NA",
				CountryCode: "US",
			},
		},
		{
			ip: "2001:470:1:43b:1::68",  // leaf.dragonflybsd.org
			location: &Location{
				ContinentCode: "NA",
				CountryCode: "US",
			},
		},
		{
			ip: "202.120.2.119",  // www.sjtu.edu.cn
			location: &Location{
				ContinentCode: "AS",
				CountryCode: "CN",
			},
		},
		{
			ip: "2001:da8:8000:1::2:119",  // www.sjtu.edu.cn
			location: &Location{
				ContinentCode: "AS",
				CountryCode: "CN",
			},
		},
	}

	for _, tc := range cases {
		ip := net.ParseIP(tc.ip)
		loc, _ := LookupIP(ip)
		if loc == nil {
			t.Errorf("LookupIP(%v) = %v, want %v", ip, loc, tc.location)
		}
		if loc.ContinentCode != tc.location.ContinentCode ||
		   loc.CountryCode != tc.location.CountryCode {
			t.Errorf("LookupIP(%v) = %v, want %v", ip, loc, tc.location)
		}
	}
}


func TestFindMirrors(t *testing.T) {
	cases := []struct {
		location *Location
		mirror_count int
		mirror_first string  // name of mirror
		mirror_last string
	}{
		{
			location: nil,
			mirror_count: 1,
			mirror_first: "SJTUG",
			mirror_last: "SJTUG",
		},
		{
			// leaf.dragonflybsd.org (199.233.90.68)
			location: &Location{
				ContinentCode: "NA",
				CountryCode: "US",
				Latitude: 37.8817,
				Longitude: -122.188,
			},
			mirror_count: 2,
			mirror_first: "DragonFly/Avalon",
			mirror_last: "SJTUG",
		},
		{
			// www.sjtu.edu.cn (202.120.2.119)
			location: &Location{
				ContinentCode: "AS",
				CountryCode: "CN",
				Latitude: 31.201,
				Longitude: 121.433,
			},
			mirror_count: 2,
			mirror_first: "SJTUG",
			mirror_last: "SJTUG",
		},
	}

	for _, tc := range cases {
		mirrors := FindMirrors(tc.location)
		if len(mirrors) != tc.mirror_count {
			t.Errorf("FindMirrors(%v) failed: got %d mirrors, want %d",
					tc.location, len(mirrors), tc.mirror_count)
		}
		m_first := mirrors[0]
		if m_first.Name != tc.mirror_first {
			t.Errorf("FindMirrors(%v) failed: got [%s], want [%s]",
					tc.location, m_first.Name, tc.mirror_first)
		}
		m_last := mirrors[len(mirrors)-1]
		if m_last.Name != tc.mirror_last {
			t.Errorf("FindMirrors(%v) failed: got [%s], want [%s]",
					tc.location, m_last.Name, tc.mirror_last)
		}
	}
}


func TestGreatCircleDistance(t *testing.T) {
	eps := 1e-6
	cases := []struct {
		p1 Point
		p2 Point
		want_min float64
		want_max float64
	}{
		{
			p1: Point{ 0, 0 },
			p2: Point{ 0, 0 },
			want_min: -eps,
			want_max: eps,
		},
		{
			p1: Point{ 0, 0 },
			p2: Point{ 90, 0 },
			want_min: 90 - eps,
			want_max: 90 + eps,
		},
		{
			p1: Point{ 0, 0 },
			p2: Point{ 180, 0 },
			want_min: 180 - eps,
			want_max: 180 + eps,
		},
		{
			p1: Point{ 0, 0 },
			p2: Point{ 0, 90 },
			want_min: 90 - eps,
			want_max: 90 + eps,
		},
		{
			p1: Point{ 0, -90 },
			p2: Point{ 0, 90 },
			want_min: 180 - eps,
			want_max: 180 + eps,
		},
	}

	for _, tc := range cases {
		rad := greatCircleDistance(&tc.p1, &tc.p2)
		deg := rad * 180 / math.Pi
		if deg < tc.want_min || deg > tc.want_max {
			t.Errorf("greatCircleDistance(%v, %v) = %v; want: [%v, %v]",
					tc.p1, tc.p2, deg, tc.want_min, tc.want_max)
		}
	}
}
