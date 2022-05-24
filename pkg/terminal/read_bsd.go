// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (darwin || freebsd || netbsd || openbsd || dragonfly) && !appengine
// +build darwin freebsd netbsd openbsd dragonfly
// +build !appengine

// https://github.com/golang/term/blob/e5f449aeb1717c7a4156811d5d65510fee9d1530/term_unix_bsd.go
package terminal

import "syscall"

const ioctlReadTermios = syscall.TIOCGETA
const ioctlWriteTermios = syscall.TIOCSETA
