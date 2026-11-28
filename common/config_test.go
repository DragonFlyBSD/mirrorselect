// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2021 The DragonFly Project.

package common

import (
	"log/slog"
	"testing"
)

func TestReadConfig(t *testing.T) {
	fname := "../testdata/mirrorselect.toml"
	cfg := ReadConfig(fname, slog.Default())

	if cfg.Monitor.TLSVerify != false {
		t.Errorf("ReadConfig(%q) failed: TLSVerify = %v, want %v\n",
			fname, cfg.Monitor.TLSVerify, false)
	}

	if len(cfg.Mirrors) != 2 {
		t.Errorf("ReadConfig(%q) failed: got %d mirrors, want %d\n",
			fname, len(cfg.Mirrors), 2)
	}

	for name, mirror := range cfg.Mirrors {
		if mirror.Status.Online != true {
			t.Errorf("ReadConfig(%q) failed: mirror [%s] status != online\n",
				fname, name)
		}
	}
}
