// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2021-2026 The DragonFly Project.

package monitor

import (
	"bufio"
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DragonFlyBSD/mirrorselect/common"
)

func writeCertPEM(t *testing.T, der []byte) string {
	t.Helper()

	fname := filepath.Join(t.TempDir(), "cert.pem")
	data := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(fname, data, 0644); err != nil {
		t.Fatalf("Failed to write certificate: %v", err)
	}
	return fname
}

func closedTcpURL(t *testing.T, scheme string) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("Failed to close listener: %v", err)
	}
	return scheme + "://" + addr
}

func startTestFtpServer(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	t.Cleanup(func() {
		ln.Close()
	})

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveTestFtpConn(conn)
		}
	}()

	return "ftp://" + ln.Addr().String()
}

// Implement just enough of the FTP protocol for ftpCheck.
func serveTestFtpConn(conn net.Conn) {
	defer conn.Close()

	fmt.Fprint(conn, "220 Test FTP server ready.\r\n")
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		fields := strings.SplitN(strings.TrimSpace(line), " ", 2)
		switch strings.ToUpper(fields[0]) {
		case "USER":
			fmt.Fprint(conn, "331 Password required.\r\n")
		case "PASS":
			fmt.Fprint(conn, "230 Logged in.\r\n")
		case "FEAT":
			fmt.Fprint(conn, "211 End\r\n")
		case "TYPE":
			fmt.Fprint(conn, "200 Type set.\r\n")
		case "CWD":
			if len(fields) != 2 || strings.Contains(fields[1], "missing") {
				fmt.Fprint(conn, "550 No such directory.\r\n")
				return
			}
			fmt.Fprint(conn, "250 Directory changed.\r\n")
		case "QUIT":
			fmt.Fprint(conn, "221 Bye.\r\n")
			return
		default:
			fmt.Fprint(conn, "500 Command not implemented.\r\n")
		}
	}
}

func TestHttpCheck(t *testing.T) {
	oldTLSVerify := appConfig.Monitor.TLSVerify
	oldTimeout := appConfig.Monitor.Timeout
	t.Cleanup(func() {
		appConfig.Monitor.TLSVerify = oldTLSVerify
		appConfig.Monitor.Timeout = oldTimeout
	})
	appConfig.Monitor.Timeout = 2

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	httpSrv := httptest.NewServer(handler)
	defer httpSrv.Close()
	tlsSrv := httptest.NewTLSServer(handler)
	defer tlsSrv.Close()

	// Trust the test server's self-signed certificate when verifying TLS.
	t.Setenv("SSL_CERT_FILE", writeCertPEM(t, tlsSrv.Certificate().Raw))

	okURLs := []string{
		httpSrv.URL + "/",
		tlsSrv.URL + "/",
		httpSrv.URL + "/BSD/DragonFlyBSD/dports",
	}
	for _, tlsVerify := range []bool{true, false} {
		appConfig.Monitor.TLSVerify = tlsVerify
		for _, utext := range okURLs {
			u, _ := url.Parse(utext)
			status, err := httpCheck(context.Background(), u)
			if err != nil || !status {
				t.Errorf("httpCheck(%q) = (%v, %v); want %v\n",
					u, status, err, true)
			}
		}
	}

	failURLs := []string{
		closedTcpURL(t, "http") + "/",
		httpSrv.URL + "/missing",
	}
	for _, utext := range failURLs {
		u, _ := url.Parse(utext)
		status, err := httpCheck(context.Background(), u)
		if err == nil || status {
			t.Errorf("httpCheck(%q) = (%v, %v); want %v\n",
				u, status, err, false)
		}
	}

	invalidURLs := []string{
		"ftp://127.0.0.1/",
		"xxx://www.example.com/",
	}
	for _, utext := range invalidURLs {
		u, _ := url.Parse(utext)
		status, err := httpCheck(context.Background(), u)
		if err == nil || status {
			t.Errorf("httpCheck(%q) = (%v, %v); want %v\n",
				u, status, err, false)
		}
	}
}

func TestFtpCheck(t *testing.T) {
	oldTimeout := appConfig.Monitor.Timeout
	t.Cleanup(func() {
		appConfig.Monitor.Timeout = oldTimeout
	})
	appConfig.Monitor.Timeout = 2

	ftpURL := startTestFtpServer(t)
	okURLs := []string{
		ftpURL + "/",
		ftpURL + "/pub/FreeBSD/",
	}
	for _, utext := range okURLs {
		u, _ := url.Parse(utext)
		status, err := ftpCheck(context.Background(), u)
		if err != nil || !status {
			t.Errorf("ftpCheck(%q) = (%v, %v); want %v\n",
				u, status, err, true)
		}
	}

	failURLs := []string{
		closedTcpURL(t, "ftp") + "/",
		ftpURL + "/missing/",
	}
	for _, utext := range failURLs {
		u, _ := url.Parse(utext)
		status, err := ftpCheck(context.Background(), u)
		if err == nil || status {
			t.Errorf("ftpCheck(%q) = (%v, %v); want %v\n",
				u, status, err, false)
		}
	}

	invalidURLs := []string{
		"http://127.0.0.1/",
		"xxx://www.example.com/",
	}
	for _, utext := range invalidURLs {
		u, _ := url.Parse(utext)
		status, err := ftpCheck(context.Background(), u)
		if err == nil || status {
			t.Errorf("ftpCheck(%q) = (%v, %v); want %v\n",
				u, status, err, false)
		}
	}
}

func TestHysteresis(t *testing.T) {
	oldHysteresis := appConfig.Monitor.Hysteresis
	t.Cleanup(func() {
		appConfig.Monitor.Hysteresis = oldHysteresis
	})
	appConfig.Monitor.Hysteresis = 2

	mirror := &common.Mirror{
		Name: "Test",
		Status: common.MirrorStatus{
			Online:     true,
			Hysteresis: 0,
		},
	}

	assertStatus := func(hysteresis int, online bool) {
		if mirror.Status.Hysteresis != hysteresis {
			t.Errorf("updateMirror() failed: hysteresis = %d; want %d\n",
				mirror.Status.Hysteresis, hysteresis)
		}
		if mirror.Status.Online != online {
			t.Errorf("updateMirror() failed: online = %v; want %v\n",
				mirror.Status.Online, online)
		}
	}

	updateMirror(context.Background(), "test", mirror, true)
	assertStatus(0, true)
	updateMirror(context.Background(), "test", mirror, false)
	assertStatus(1, true)
	updateMirror(context.Background(), "test", mirror, false)
	assertStatus(0, false)
	updateMirror(context.Background(), "test", mirror, false)
	assertStatus(0, false)
	updateMirror(context.Background(), "test", mirror, true)
	assertStatus(1, false)
	updateMirror(context.Background(), "test", mirror, false)
	assertStatus(0, false)
	updateMirror(context.Background(), "test", mirror, true)
	assertStatus(1, false)
	updateMirror(context.Background(), "test", mirror, true)
	assertStatus(0, true)
}

func TestStartMonitorContextCancel(t *testing.T) {
	oldMirrors := appConfig.Mirrors
	oldWorkers := appConfig.Monitor.Workers
	oldInterval := appConfig.Monitor.Interval
	oldTimeout := appConfig.Monitor.Timeout
	oldHysteresis := appConfig.Monitor.Hysteresis
	oldNotifyExec := appConfig.Monitor.NotifyExec
	t.Cleanup(func() {
		appConfig.Mirrors = oldMirrors
		appConfig.Monitor.Workers = oldWorkers
		appConfig.Monitor.Interval = oldInterval
		appConfig.Monitor.Timeout = oldTimeout
		appConfig.Monitor.Hysteresis = oldHysteresis
		appConfig.Monitor.NotifyExec = oldNotifyExec
	})
	appConfig.Monitor.Workers = 2
	appConfig.Monitor.Interval = 3600
	appConfig.Monitor.Timeout = 30
	appConfig.Monitor.Hysteresis = 1
	appConfig.Monitor.NotifyExec = ""

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case started <- struct{}{}:
		default:
		}
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() {
		close(release)
	})

	mirror := &common.Mirror{
		Name: "Test",
		URL:  srv.URL + "/",
		Status: common.MirrorStatus{
			Online: true,
		},
	}
	appConfig.Mirrors = map[string]*common.Mirror{"test": mirror}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		StartMonitor(ctx, slog.Default())
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("Monitor did not start a check")
	}
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("StartMonitor did not stop after context cancellation")
	}

	if mirror.Status.OKCount != 0 || mirror.Status.ErrorCount != 0 {
		t.Errorf("Mirror status changed after cancellation: %+v", mirror.Status)
	}
}

func TestHttpCheckContextCancel(t *testing.T) {
	oldTimeout := appConfig.Monitor.Timeout
	t.Cleanup(func() {
		appConfig.Monitor.Timeout = oldTimeout
	})
	appConfig.Monitor.Timeout = 30

	started := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() {
		close(release)
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		u, _ := url.Parse(srv.URL + "/")
		status, err := httpCheck(ctx, u)
		if status {
			err = fmt.Errorf("status = true")
		}
		done <- err
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("HTTP check did not start")
	}
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("httpCheck() error = %v; want %v", err, context.Canceled)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("httpCheck did not return after context cancellation")
	}
}

func TestFtpCheckContextCancel(t *testing.T) {
	oldTimeout := appConfig.Monitor.Timeout
	t.Cleanup(func() {
		appConfig.Monitor.Timeout = oldTimeout
	})
	appConfig.Monitor.Timeout = 30

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	t.Cleanup(func() {
		ln.Close()
	})

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := ln.Accept()
		if err == nil {
			accepted <- conn
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		u, _ := url.Parse("ftp://" + ln.Addr().String())
		status, err := ftpCheck(ctx, u)
		if status {
			err = fmt.Errorf("status = true")
		}
		done <- err
	}()

	select {
	case conn := <-accepted:
		t.Cleanup(func() {
			conn.Close()
		})
		cancel()
	case <-time.After(time.Second):
		t.Fatal("FTP check did not connect")
	}

	select {
	case err := <-done:
		if err == nil {
			t.Error("ftpCheck() succeeded after context cancellation")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ftpCheck did not return after context cancellation")
	}
}

func TestCheckMirrorContextCancel(t *testing.T) {
	oldTimeout := appConfig.Monitor.Timeout
	t.Cleanup(func() {
		appConfig.Monitor.Timeout = oldTimeout
	})
	appConfig.Monitor.Timeout = 30

	started := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() {
		close(release)
	})

	mirror := &common.Mirror{
		Name: "Test",
		URL:  srv.URL + "/",
		Status: common.MirrorStatus{
			Online: true,
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		checkMirror(ctx, "test", mirror)
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("Mirror check did not start")
	}
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("checkMirror did not return after context cancellation")
	}

	if mirror.Status.OKCount != 0 || mirror.Status.ErrorCount != 0 {
		t.Errorf("Mirror status changed after cancellation: %+v", mirror.Status)
	}
	if !mirror.Status.Online {
		t.Errorf("Mirror went offline after cancellation: %+v", mirror.Status)
	}
}

func TestNotifyExecContextCancel(t *testing.T) {
	oldNotifyExec := appConfig.Monitor.NotifyExec
	oldExecTimeout := appConfig.Monitor.ExecTimeout
	t.Cleanup(func() {
		appConfig.Monitor.NotifyExec = oldNotifyExec
		appConfig.Monitor.ExecTimeout = oldExecTimeout
	})
	appConfig.Monitor.ExecTimeout = 30

	dir := t.TempDir()
	marker := filepath.Join(dir, "started")
	script := filepath.Join(dir, "notify.sh")
	err := os.WriteFile(script,
		[]byte("#!/bin/sh\n: > \"$MARKER\"\nexec sleep 30\n"), 0755)
	if err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}
	appConfig.Monitor.NotifyExec = script
	t.Setenv("MARKER", marker)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		notifyExec(ctx, appConfig.Monitor, "test", true)
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Notification command did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("notifyExec did not return after context cancellation")
	}
}
