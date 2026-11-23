// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2021 The DragonFly Project.

package common

const AppName = "mirrorselect"

// Use 'var' instead of 'const' since we may override them via -ldflags.
var Version = "$Format:%(describe:tags=true,abbrev=0)$"
var Commit = "$Format:%h$"
var Date = "$Format:%cs$"
