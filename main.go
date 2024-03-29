//
// DragonFly pkg mirrorselect
//

//
// Copyright (c) 2021 The DragonFly Project.
//
// This code is derived from software contributed to The DragonFly Project
// by Aaron LI <aly@aaronly.me>.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions
// are met:
//
// 1. Redistributions of source code must retain the above copyright
//    notice, this list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright
//    notice, this list of conditions and the following disclaimer in
//    the documentation and/or other materials provided with the
//    distribution.
// 3. Neither the name of The DragonFly Project nor the names of its
//    contributors may be used to endorse or promote products derived
//    from this software without specific, prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// ``AS IS'' AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS
// FOR A PARTICULAR PURPOSE ARE DISCLAIMED.  IN NO EVENT SHALL THE
// COPYRIGHT HOLDERS OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT,
// INCIDENTAL, SPECIAL, EXEMPLARY OR CONSEQUENTIAL DAMAGES (INCLUDING,
// BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED
// AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
// OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT
// OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
// SUCH DAMAGE.
//

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/user"

	"github.com/gin-gonic/gin"

	"github.com/DragonFlyBSD/mirrorselect/api"
	"github.com/DragonFlyBSD/mirrorselect/common"
	"github.com/DragonFlyBSD/mirrorselect/monitor"
)

func main() {
	var cfgfile string
	var accesslog string
	var f_version bool
	flag.StringVar(&cfgfile, "config", common.AppName+".toml", "config file")
	flag.StringVar(&accesslog, "access-log", "", "web access log file")
	flag.BoolVar(&f_version, "version", false, "show version")
	flag.Parse()

	if f_version {
		fmt.Printf("Version: %s\n", common.Version)
		fmt.Printf("Commit: %s\n", common.Commit)
		fmt.Printf("Date: %s\n", common.Date)
		return
	}

	if u, _ := user.Current(); u.Uid == "0" {
		common.WarnPrintf("Running as root (uid=0) is discouraged!!!")
	}

	cfg := common.ReadConfig(cfgfile)

	gin.SetMode(gin.ReleaseMode)
	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
	}

	if accesslog != "" {
		f, err := os.OpenFile(accesslog,
				os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			common.Fatalf("Failed to open access log: %v\n", err)
		}
		gin.DisableConsoleColor()
		gin.DefaultWriter = io.MultiWriter(f)
		common.InfoPrintf("Write access log to file: %s\n", accesslog)
	}

	router := gin.Default()
	router.GET("/", api.GetPing)
	router.GET("/pkg/:abi/*path", api.GetPkgMirrors)
	router.GET("/mirror", api.GetMirrors)
	router.GET("/mirrors", api.GetMirrors)
	router.GET("/ip", api.GetIP)
	router.GET("/ping", api.GetPing)

	go monitor.StartMonitor()

	common.InfoPrintf("Listen on: [%s]\n", cfg.Listen)
	router.Run(cfg.Listen)
}
