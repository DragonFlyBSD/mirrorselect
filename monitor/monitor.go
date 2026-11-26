// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2021-2026 The DragonFly Project.

package monitor

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
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
// It runs until the context is canceled, then stops the worker pool.
func StartMonitor(ctx context.Context) {
	common.InfoPrintf("Start mirror monitor.\n")

	tasks := make([]*workerpool.Task, 0, len(appConfig.Mirrors))
	for name, mirror := range appConfig.Mirrors {
		// NOTE: Need to make a copy of the loop variables
		n := name
		m := mirror
		f := func(ctx context.Context, _ any) error {
			checkMirror(ctx, n, m)
			return nil
		}
		tasks = append(tasks, workerpool.NewTask(f, nil))
	}

	pool := workerpool.NewPool(tasks, appConfig.Monitor.Workers)
	poolDone := make(chan struct{})
	go func() {
		pool.Run(ctx)
		close(poolDone)
	}()

	defer func() {
		pool.Stop()
		<-poolDone
	}()

	ticker := time.NewTicker(time.Duration(appConfig.Monitor.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			common.InfoPrintf("Stop mirror monitor.\n")
			return
		case <-ticker.C:
			for i := range tasks {
				if !pool.AddTask(ctx, tasks[i]) {
					break
				}
			}
		}
	}
}

// Check the given mirror and update its status.
func checkMirror(ctx context.Context, name string, mirror *common.Mirror) {
	u, err := url.Parse(mirror.URL)
	if err != nil {
		common.Fatalf("Mirror [%s] URL invalid: %v\n",
			name, mirror.URL)
	}

	status := false
	switch u.Scheme {
	case "http", "https":
		status, err = httpCheck(ctx, u)
	case "ftp":
		status, err = ftpCheck(ctx, u)
	default:
		common.Fatalf("Mirror [%s] URL unsupported: %v\n",
			name, mirror.URL)
	}
	common.DebugPrintf("Mirror [%s]: %v, error: %v\n",
		name, status, err)

	// Don't update the status if the check was interrupted by shutdown.
	if ctx.Err() != nil {
		return
	}
	updateMirror(ctx, name, mirror, status)
}

// Update the status of a mirror accodring to the check result.
func updateMirror(ctx context.Context, name string, mirror *common.Mirror, status bool) {
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
			go notifyExec(ctx, name, status)
		}
	} else {
		mirror.Status.Hysteresis = 0
		common.DebugPrintf("Mirror [%s] hysteresis = %d\n",
			name, mirror.Status.Hysteresis)
	}
}

// Check the given HTTP/HTTPS URL to determine whether it's accessible.
func httpCheck(ctx context.Context, u *url.URL) (bool, error) {
	if u.Scheme != "http" && u.Scheme != "https" {
		return false, fmt.Errorf("Invalid HTTP(s) URL: %v", u.String())
	}

	timeout := time.Duration(appConfig.Monitor.Timeout) * time.Second
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: !appConfig.Monitor.TLSVerify,
		ServerName:         u.Hostname(),
	}
	client := &http.Client{
		Timeout:   timeout,
		Transport: tr,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
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

// cancelConn closes the connection when the context is canceled.
// The FTP library only uses the context for dialing, so closing the
// connection is needed to interrupt pending reads from an unresponsive server.
type cancelConn struct {
	net.Conn
	stop func() bool
}

func (c *cancelConn) Close() error {
	c.stop()
	return c.Conn.Close()
}

// Check the given FTP URL to determine whether it's accessible.
func ftpCheck(ctx context.Context, u *url.URL) (bool, error) {
	if u.Scheme != "ftp" {
		return false, fmt.Errorf("Invalid FTP URL: %v", u.String())
	}

	addr := u.Host
	if u.Port() == "" {
		addr += ":21"
	}

	timeout := time.Duration(appConfig.Monitor.Timeout) * time.Second
	dialer := net.Dialer{Timeout: timeout}
	dialFunc := func(network, address string) (net.Conn, error) {
		conn, err := dialer.DialContext(ctx, network, address)
		if err != nil {
			return nil, err
		}
		return &cancelConn{
			Conn: conn,
			stop: context.AfterFunc(ctx, func() { conn.Close() }),
		}, nil
	}
	conn, err := ftp.Dial(addr, ftp.DialWithDialFunc(dialFunc))
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
func notifyExec(ctx context.Context, name string, status bool) {
	if appConfig.Monitor.NotifyExec == "" {
		return
	}

	event := "DOWN"
	if status {
		event = "UP"
	}

	timeout := time.Duration(appConfig.Monitor.ExecTimeout) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, appConfig.Monitor.NotifyExec, name, event)
	common.DebugPrintf("Command: %s\n", cmd.String())
	output, err := cmd.CombinedOutput()
	if output != nil && len(output) > 0 {
		common.InfoPrintf("Command output: %s", output)
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		common.ErrorPrintf("Command (%s) timed out!\n", cmd.String())
	} else if errors.Is(ctx.Err(), context.Canceled) {
		common.DebugPrintf("Command (%s) canceled.\n", cmd.String())
	} else if err != nil {
		common.ErrorPrintf("Command (%s) failed: %v\n", cmd.String(), err)
	}
}
