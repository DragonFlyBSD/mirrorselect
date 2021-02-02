package api

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/DragonFlyBSD/mirrorselect/common"
	"github.com/DragonFlyBSD/mirrorselect/geoip"
)

var appConfig = common.AppConfig


// A demo that simply responses the request.
//
func GetPing(c *gin.Context) {
	c.String(http.StatusOK, "pong\n")
}

// Return the IP and location information about client.
//
func GetIP(c *gin.Context) {
	ip := net.ParseIP(c.ClientIP())
	if ip == nil {
		common.DebugPrintf("Invalid client IP: %s\n", c.ClientIP())
		c.String(http.StatusBadRequest, "Invalid client IP!\n")
		return
	}

	location, err := geoip.LookupIP(ip)
	if err != nil {
		common.DebugPrintf("Lookup IP (%s) error: %v\n", ip.String(), err)
	}

	info := fmt.Sprintf("IP: %s\n", ip.String())
	if location == nil {
		info += fmt.Sprintf("Location: unknown\n")
	} else {
		info += fmt.Sprintf("Location:\nContinent: %s\nCountry: %s\n",
				location.ContinentCode, location.CountryCode)
		info += fmt.Sprintf("Latitude: %v\nLongitude: %v\n",
				location.Latitude, location.Longitude)
	}
	c.String(http.StatusOK, info)
}

// Return current status of all mirrors.
//
func GetMirrors(c *gin.Context) {
	c.JSON(http.StatusOK, appConfig.Mirrors)
}

// Return mirrors based on the client's location.
//
func GetPkgMirrors(c *gin.Context) {
	ip := net.ParseIP(c.ClientIP())
	if ip == nil {
		common.DebugPrintf("Invalid client IP: %s\n", c.ClientIP())
		c.String(http.StatusBadRequest, "Invalid client IP!\n")
		return
	}

	location, err := geoip.LookupIP(ip)
	if err != nil {
		common.DebugPrintf("Lookup IP (%s) error: %v\n", ip.String(), err)
	}
	common.DebugPrintf("Client IP: %s, Location: %v\n", ip.String(), location)

	mirrors := geoip.FindMirrors(location)
	urls := ""
	for _, m := range mirrors {
		urls += fmt.Sprintf("URL: %s/%s/%s\n", m.URL, c.Param("abi"),
				strings.TrimPrefix(c.Param("path"), "/"))
	}
	c.String(http.StatusOK, urls)
}
