// Copyright (c) 2012-2015, Sergey Cherepanov
// All rights reserved.
// Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:
// * Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
// * Neither the name of the author nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

//go:build windows
// +build windows

// Taken from https://github.com/cheggaaa/pb/blob/bbc97ac4868838de454ce23dac35e7d6f06db533/v3/termutil/term_win.go
package terminal

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

var tty = os.Stdin

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	// GetConsoleMode retrieves the current input mode of a console's
	// input buffer or the current output mode of a console screen buffer.
	// https://msdn.microsoft.com/en-us/library/windows/desktop/ms683167(v=vs.85).aspx
	getConsoleMode = kernel32.NewProc("GetConsoleMode")

	// SetConsoleMode sets the input mode of a console's input buffer
	// or the output mode of a console screen buffer.
	// https://msdn.microsoft.com/en-us/library/windows/desktop/ms686033(v=vs.85).aspx
	setConsoleMode = kernel32.NewProc("SetConsoleMode")
)

type (
	// Defines the coordinates of the upper left and lower right corners
	// of a rectangle.
	// See
	// http://msdn.microsoft.com/en-us/library/windows/desktop/ms686311(v=vs.85).aspx
	smallRect struct {
		Left, Top, Right, Bottom int16
	}

	// Defines the coordinates of a character cell in a console screen
	// buffer. The origin of the coordinate system (0,0) is at the top, left cell
	// of the buffer.
	// See
	// http://msdn.microsoft.com/en-us/library/windows/desktop/ms682119(v=vs.85).aspx
	coordinates struct {
		X, Y int16
	}

	word int16
)

var ErrPoolWasStarted = errors.New("Bar pool was started")

var echoLocked bool
var echoLockMutex sync.Mutex

var oldState word

func Lock() (shutdownCh chan struct{}, err error) {
	echoLockMutex.Lock()
	defer echoLockMutex.Unlock()
	if echoLocked {
		err = ErrPoolWasStarted
		return
	}
	echoLocked = true

	if _, _, e := syscall.Syscall(getConsoleMode.Addr(), 2, uintptr(syscall.Stdout), uintptr(unsafe.Pointer(&oldState)), 0); e != 0 {
		err = fmt.Errorf("Can't get terminal settings: %v", e)
		return
	}

	newState := oldState
	const ENABLE_ECHO_INPUT = 0x0004
	const ENABLE_LINE_INPUT = 0x0002
	newState = newState & (^(ENABLE_LINE_INPUT | ENABLE_ECHO_INPUT))
	if _, _, e := syscall.Syscall(setConsoleMode.Addr(), 2, uintptr(syscall.Stdout), uintptr(newState), 0); e != 0 {
		err = fmt.Errorf("Can't set terminal settings: %v", e)
		return
	}

	shutdownCh = make(chan struct{})
	return
}

func Unlock() (err error) {
	echoLockMutex.Lock()
	defer echoLockMutex.Unlock()
	if !echoLocked {
		return
	}
	echoLocked = false
	if _, _, e := syscall.Syscall(setConsoleMode.Addr(), 2, uintptr(syscall.Stdout), uintptr(oldState), 0); e != 0 {
		err = fmt.Errorf("Can't set terminal settings")
	}
	return
}
