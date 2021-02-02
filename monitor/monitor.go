package monitor

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
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
		var ok bool
		var err error

		if strings.HasPrefix(mirror.URL, "http://") ||
		   strings.HasPrefix(mirror.URL, "https://") {
			ok, err = httpCheck(mirror.URL)
		} else if strings.HasPrefix(mirror.URL, "ftp://") {
			// TODO
		} else {
			ok = false
			err = fmt.Errorf("Mirror [%s] has invalid URL: %v",
				name, mirror.URL)
		}
		common.DebugPrintf("Mirror [%s]: %v, error: %v\n", name, ok, err)

		updateMirror(name, mirror, ok)
	}
}


// Update the status of a mirror accodring to the check result.
//
func updateMirror(name string, mirror *common.Mirror, ok bool) {
	if ok {
		mirror.Status.OKCount++
	} else {
		mirror.Status.ErrorCount++
	}

	if mirror.Status.Offline == ok {
		mirror.Status.Hysteresis++
		common.DebugPrintf("Mirror [%s] hysteresis = %d\n",
				name, mirror.Status.Hysteresis)
		if mirror.Status.Hysteresis >= appConfig.Monitor.Hysteresis {
			mirror.Status.Hysteresis = 0
			mirror.Status.Offline = !ok
			if ok {
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


// Check the given HTTP URL to determine whether it's accessible.
//
func httpCheck(url string) (bool, error) {
	if !strings.HasPrefix(url, "http://") &&
	   !strings.HasPrefix(url, "https://") {
		return false, fmt.Errorf("Invalid HTTP(s) URL: %v", url)
	}
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}

	var tr *http.Transport
	if strings.HasPrefix(url, "https:") && !appConfig.Monitor.TLSVerify {
		tr = http.DefaultTransport.(*http.Transport).Clone()
		tr.TLSClientConfig = &tls.Config{ InsecureSkipVerify: true }
	}

	client := &http.Client{
		Transport: tr,
		Timeout: appConfig.Monitor.Timeout * time.Second,
	}

	resp, err := client.Get(url)
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
