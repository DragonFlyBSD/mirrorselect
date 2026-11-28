// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2021 The DragonFly Project.

package api

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/DragonFlyBSD/mirrorselect/common"
	"github.com/DragonFlyBSD/mirrorselect/geoip"
)

var appConfig = common.AppConfig
var logger = slog.Default()

// A demo that simply responses the request.
func GetPing(c *gin.Context) {
	c.String(http.StatusOK, "pong\n")
}

// Return the IP and location information about client.
func GetIP(c *gin.Context) {
	ip := net.ParseIP(c.ClientIP())
	if ip == nil {
		logger.Debug("invalid client", "ip", c.ClientIP())
		c.String(http.StatusBadRequest, "Invalid client IP!\n")
		return
	}

	location, err := geoip.LookupIP(ip)
	if err != nil {
		logger.Debug("failed to lookup IP", "ip", ip.String(), "error", err)
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
func GetMirrors(c *gin.Context) {
	c.JSON(http.StatusOK, appConfig.Mirrors)
}

// Return mirrors based on the client's location.
func GetPkgMirrors(c *gin.Context) {
	ip := net.ParseIP(c.ClientIP())
	if ip == nil {
		logger.Debug("invalid client", "ip", c.ClientIP())
		c.String(http.StatusBadRequest, "Invalid client IP!\n")
		return
	}

	location, err := geoip.LookupIP(ip)
	if err != nil {
		logger.Debug("failed to lookup IP", "ip", ip.String(), "error", err)
	}
	logger.Debug("lookup result", "ip", ip.String(), "location", location)

	mirrors := geoip.FindMirrors(location)
	urls := ""
	for _, m := range mirrors {
		urls += fmt.Sprintf("URL: %s/%s/%s\n",
			strings.TrimSuffix(m.URL, "/"), c.Param("abi"),
			strings.TrimPrefix(c.Param("path"), "/"))
	}
	c.String(http.StatusOK, urls)
}
