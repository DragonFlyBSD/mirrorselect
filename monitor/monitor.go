package monitor

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"time"

	"github.com/jlaffaye/ftp"

	"github.com/DragonFlyBSD/mirrorselect/common"
	"github.com/DragonFlyBSD/mirrorselect/workerpool"
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
	var tasks []*workerpool.Task

	for name, mirror := range appConfig.Mirrors {
		// NOTE: Need to make a copy of the loop variables
		n := name
		m := mirror
		f := func(data interface{}) error {
			checkMirror(n, m)
			return nil
		}
		tasks = append(tasks, workerpool.NewTask(f, nil))
	}

	pool := workerpool.NewPool(tasks, appConfig.Monitor.Workers)
	pool.Run()
}


// Check the given mirror and update its status.
//
func checkMirror(name string, mirror *common.Mirror) {
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
		status, err = ftpCheck(u)
	default:
		common.Fatalf("Mirror [%s] URL unsupported: %v\n",
				name, mirror.URL)
	}
	common.DebugPrintf("Mirror [%s]: %v, error: %v\n",
			name, status, err)

	updateMirror(name, mirror, status)
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
			go notifyExec(name, status)
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

	timeout := appConfig.Monitor.Timeout * time.Second
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: !appConfig.Monitor.TLSVerify,
		ServerName: u.Hostname(),
	}
	client := &http.Client{
		Timeout: timeout,
		Transport: tr,
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return false, err
	}

	req.Host = u.Host
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", appConfig.Monitor.UserAgent)

	resp, err := client.Do(req)
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


// Check the given FTP URL to determine whether it's accessible.
//
func ftpCheck(u *url.URL) (bool, error) {
	if u.Scheme != "ftp" {
		return false, fmt.Errorf("Invalid FTP URL: %v", u.String())
	}

	addr := u.Host
	if u.Port() == "" {
		addr += ":21"
	}

	timeout := appConfig.Monitor.Timeout * time.Second
	conn, err := ftp.Dial(addr, ftp.DialWithTimeout(timeout))
	if err != nil {
		return false, err
	}

	err = conn.Login("anonymous", "anonymous")
	if err != nil {
		return false, err
	}
	err = conn.ChangeDir(u.Path)
	if err != nil {
		return false, err
	}

	err = conn.Quit()
	if err != nil {
		return false, err
	}

	return true, nil
}


// Publish the mirror event by invoking the configured notification
// executable.
//
func notifyExec(name string, status bool) {
	if appConfig.Monitor.NotifyExec == "" {
		return
	}

	event := "DOWN"
	if status {
		event = "UP"
	}

	timeout := appConfig.Monitor.ExecTimeout * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, appConfig.Monitor.NotifyExec, name, event)
	common.DebugPrintf("Command: %s\n", cmd.String())
	output, err := cmd.CombinedOutput()
	if output != nil && len(output) > 0 {
		common.InfoPrintf("Command output: %s", output)
	}
	if ctx.Err() != nil {
		common.ErrorPrintf("Command (%s) timed out!\n", cmd.String())
	} else if err != nil {
		common.ErrorPrintf("Command (%s) failed: %v\n", cmd.String(), err)
	}
}
