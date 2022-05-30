// Copyright (c) 2012-2015, Sergey Cherepanov
// All rights reserved.
// Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:
// * Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
// * Neither the name of the author nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
// See https://github.com/cheggaaa/pb/blob/master/v3/termutil/term_x.go

//go:build (linux || darwin || freebsd || netbsd || openbsd || solaris || dragonfly) && (!appengine || !js)
// +build linux darwin freebsd netbsd openbsd solaris dragonfly
// +build !appengine !js

package terminal

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	tty      *os.File
	oldState syscall.Termios
)

func init() {
	var err error
	tty, err = os.Open("/dev/tty")
	if err != nil {
		tty = os.Stdin
	}
}

const sysIoctl = syscall.SYS_IOCTL

func Lock() (_ chan struct{}, err error) {
	fd := tty.Fd()
	if _, _, e := syscall.Syscall6(sysIoctl, fd, ioctlReadTermios, uintptr(unsafe.Pointer(&oldState)), 0, 0, 0); e != 0 {
		err = fmt.Errorf("Can't get terminal settings: %v", e)
		return
	}

	newState := oldState
	newState.Lflag &^= syscall.ECHO
	newState.Lflag |= syscall.ICANON | syscall.ISIG
	newState.Iflag |= syscall.ICRNL
	if _, _, e := syscall.Syscall6(sysIoctl, fd, ioctlWriteTermios, uintptr(unsafe.Pointer(&newState)), 0, 0, 0); e != 0 {
		err = fmt.Errorf("Can't set terminal settings: %v", e)
		return
	}
	return
}

func Unlock() (err error) {
	fd := tty.Fd()
	if _, _, e := syscall.Syscall6(sysIoctl, fd, ioctlWriteTermios, uintptr(unsafe.Pointer(&oldState)), 0, 0, 0); e != 0 {
		err = fmt.Errorf("Can't set terminal settings")
	}
	return
}
