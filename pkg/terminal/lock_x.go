// Copyright (c) 2012-2015, Sergey Cherepanov
// All rights reserved.
// Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:
// * Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
// * Neither the name of the author nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

//go:build (linux || darwin || freebsd || netbsd || openbsd || solaris || dragonfly || aix || zos) && (!appengine || !js)
// +build linux darwin freebsd netbsd openbsd solaris dragonfly aix zos
// +build !appengine !js

package terminal

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

var (
	ErrPoolWasStarted = errors.New("Bar pool was started")
	echoLockMutex     sync.Mutex
	origTermStatePtr  *unix.Termios
	tty               *os.File
	istty             bool
)

func init() {
	echoLockMutex.Lock()
	defer echoLockMutex.Unlock()

	var err error
	tty, err = os.Open("/dev/tty")
	istty = true
	if err != nil {
		tty = os.Stdin
		istty = false
	}
}

func Lock() (chan struct{}, error) {
	echoLockMutex.Lock()
	defer echoLockMutex.Unlock()
	if istty {
		if origTermStatePtr != nil {
			return nil, ErrPoolWasStarted
		}

		fd := int(tty.Fd())

		origTermStatePtr, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
		if err != nil {
			return nil, fmt.Errorf("Can't get terminal settings: %v", err)
		}

		oldTermios := *origTermStatePtr
		newTermios := oldTermios
		newTermios.Lflag &^= syscall.ECHO
		newTermios.Lflag |= syscall.ICANON | syscall.ISIG
		newTermios.Iflag |= syscall.ICRNL
		if err := unix.IoctlSetTermios(fd, ioctlWriteTermios, &newTermios); err != nil {
			return nil, fmt.Errorf("Can't set terminal settings: %v", err)
		}

	}
	shutdownCh := make(chan struct{})
	go catchTerminate(shutdownCh)
	return shutdownCh, nil
}

// listen exit signals and restore terminal state
func catchTerminate(shutdownCh chan struct{}) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGKILL)
	defer signal.Stop(sig)
	select {
	case <-shutdownCh:
		Unlock()
	case <-sig:
		Unlock()
	}
}

func Unlock() error {
	echoLockMutex.Lock()
	defer echoLockMutex.Unlock()
	if istty {
		if origTermStatePtr == nil {
			return nil
		}

		fd := int(tty.Fd())

		if err := unix.IoctlSetTermios(fd, ioctlWriteTermios, origTermStatePtr); err != nil {
			return fmt.Errorf("Can't set terminal settings: %v", err)
		}

	}
	origTermStatePtr = nil

	return nil
}
