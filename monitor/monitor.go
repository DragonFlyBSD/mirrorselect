package monitor

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"dragonflybsd/mirrorselect/common"
)

var appConfig = common.AppConfig


// Start a monitor that periodically check the status of all mirrors.
//
func StartMonitor() {
	log.Println("Start mirror monitor.")

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

		if appConfig.Debug {
			log.Printf("[DEBUG] Mirror [%s]: %v, error: %v\n",
					name, ok, err)
		}

		updateMirror(name, mirror, ok)
	}
}


// Update the status of a mirror accodring to the check result.
//
func updateMirror(name string, mirror *common.Mirror, ok bool) {
	status := map[bool]string{ true: "online", false: "offline" }

	if ok {
		mirror.Status.OKCount++
	} else {
		mirror.Status.ErrorCount++
	}

	if mirror.Status.Offline == ok {
		log.Printf("[WARNING] Mirror [%s]: %s -> %s\n", name,
				status[!ok], status[ok])
		mirror.Status.Offline = !ok
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
