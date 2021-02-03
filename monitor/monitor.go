package monitor

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/DragonFlyBSD/mirrorselect/common"
)

var appConfig = common.AppConfig


// Start a monitor that periodically check the status of all mirrors.
//
func StartMonitor() {
	common.InfoPrintf("Start mirror monitor.\n")

	for {
		checkMirrors()
		time.Sleep(appConfig.Monitor.Interval * time.Second)
	}
}


// Perform one round of check of all mirrors and update their status.
//
func checkMirrors() {
	for name, mirror := range appConfig.Mirrors {
		u, err := url.Parse(mirror.URL)
		if err != nil {
			common.Fatalf("Mirror [%s] URL invalid: %v\n",
					name, mirror.URL)
		}

		status := false
		switch u.Scheme {
		case "http", "https":
			status, err = httpCheck(u)
		case "ftp":
			// TODO
		default:
			common.Fatalf("Mirror [%s] URL unsupported: %v\n",
					name, mirror.URL)
		}
		common.DebugPrintf("Mirror [%s]: %v, error: %v\n",
				name, status, err)

		updateMirror(name, mirror, status)
	}
}


// Update the status of a mirror accodring to the check result.
//
func updateMirror(name string, mirror *common.Mirror, status bool) {
	if status {
		mirror.Status.OKCount++
	} else {
		mirror.Status.ErrorCount++
	}

	if mirror.Status.Online != status {
		mirror.Status.Hysteresis++
		common.DebugPrintf("Mirror [%s] hysteresis = %d\n",
				name, mirror.Status.Hysteresis)
		if mirror.Status.Hysteresis >= appConfig.Monitor.Hysteresis {
			mirror.Status.Hysteresis = 0
			mirror.Status.Online = status
			if status {
				common.InfoPrintf("Mirror [%s] came UP.\n", name)
			} else {
				common.WarnPrintf("Mirror [%s] went DOWN!\n", name)
			}
		}
	} else {
		mirror.Status.Hysteresis = 0
		common.DebugPrintf("Mirror [%s] hysteresis = %d\n",
				name, mirror.Status.Hysteresis)
	}
}


// Check the given HTTP/HTTPS URL to determine whether it's accessible.
//
func httpCheck(u *url.URL) (bool, error) {
	if u.Scheme != "http" && u.Scheme != "https" {
		return false, fmt.Errorf("Invalid HTTP(s) URL: %v", u.String())
	}

	var tr *http.Transport
	if u.Scheme == "https" && !appConfig.Monitor.TLSVerify {
		tr = http.DefaultTransport.(*http.Transport).Clone()
		tr.TLSClientConfig = &tls.Config{ InsecureSkipVerify: true }
	}

	client := &http.Client{
		Transport: tr,
		Timeout: appConfig.Monitor.Timeout * time.Second,
	}

	resp, err := client.Get(u.String())
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("Status code (%d) != OK",
				resp.StatusCode)
	}

	return true, nil
}
