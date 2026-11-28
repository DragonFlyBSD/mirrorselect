// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2021-2026 The DragonFly Project.

package monitor

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
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
var logger = slog.Default()

// Start a monitor that periodically check the status of all mirrors.
// It runs until the context is canceled, then stops the worker pool.
func StartMonitor(ctx context.Context, base *slog.Logger) {
	logger = base.With("component", "monitor")

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

	logger.Info("started mirror monitor")
	for {
		select {
		case <-ctx.Done():
			logger.Info("stopped mirror monitor")
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
	u, _ := url.Parse(mirror.URL)

	var online bool
	var err error
	switch u.Scheme {
	case "http", "https":
		online, err = httpCheck(ctx, u)
	case "ftp":
		online, err = ftpCheck(ctx, u)
	default:
		panic("invalid URL scheme: " + u.Scheme)
	}
	logger.Debug("checked mirror", "name", name, "online", online, "error", err)

	// Don't update the status if the check was interrupted by shutdown.
	if ctx.Err() != nil {
		return
	}
	updateMirror(ctx, name, mirror, online)
}

// Update the status of a mirror accodring to the check result.
func updateMirror(ctx context.Context, name string, mirror *common.Mirror, online bool) {
	if online {
		mirror.Status.OKCount++
	} else {
		mirror.Status.ErrorCount++
	}

	if mirror.Status.Online != online {
		mirror.Status.Hysteresis++
		logger.Debug("mirror status", "name", name, "hysteresis", mirror.Status.Hysteresis)
		if mirror.Status.Hysteresis >= appConfig.Monitor.Hysteresis {
			mirror.Status.Hysteresis = 0
			mirror.Status.Online = online
			if online {
				logger.Info("mirror changed", "name", name, "status", "UP")
			} else {
				logger.Warn("mirror changed", "name", name, "status", "DOWN!")
			}
			// Snapshot the configuration before spawning the
			// goroutine, so it doesn't access the package globals
			// to avoid data race.
			go notifyExec(ctx, appConfig.Monitor, name, online)
		}
	} else {
		mirror.Status.Hysteresis = 0
		logger.Debug("mirror status", "name", name, "hysteresis", mirror.Status.Hysteresis)
	}
}

// Check the given HTTP/HTTPS URL to determine whether it's accessible.
func httpCheck(ctx context.Context, u *url.URL) (bool, error) {
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
		return false, fmt.Errorf("status code (%d) != OK", resp.StatusCode)
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
// executable. The config and logger are passed explicitly because this
// function runs in its own goroutine.
func notifyExec(ctx context.Context, cfg common.MonitorConfig, name string, online bool) {
	if cfg.NotifyExec == "" {
		return
	}

	event := "DOWN"
	if online {
		event = "UP"
	}

	timeout := time.Duration(cfg.ExecTimeout) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cfg.NotifyExec, name, event)
	logger.Debug("executing notify", "command", cmd.String())
	output, err := cmd.CombinedOutput()
	if output != nil && len(output) > 0 {
		logger.Debug("command output", "content", output)
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		logger.Error("command timed out", "command", cmd.String())
	} else if errors.Is(ctx.Err(), context.Canceled) {
		logger.Debug("command canceled", "command", cmd.String())
	} else if err != nil {
		logger.Error("command failed", "command", cmd.String(), "error", err)
	}
}
