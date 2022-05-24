// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (linux || solaris || aix || zos) && !appengine
// +build linux solaris aix zos
// +build !appengine

// https://github.com/golang/term/blob/c04ba851c2a451287ce12942595152e208390af9/term_unix_other.go
package terminal

import "golang.org/x/sys/unix"

const ioctlReadTermios = unix.TCGETS
const ioctlWriteTermios = unix.TCSETS
