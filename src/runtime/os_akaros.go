// Copyright 2018 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import "unsafe"

func futex(addr unsafe.Pointer, op int32, val uint32, ts, addr2 unsafe.Pointer, val3 uint32) int32
func clone(flags int32, stk, mm, gg, fn unsafe.Pointer) int32
func sigaction(sig int32, new, old unsafe.Pointer)
func setitimer(mode int32, new, old unsafe.Pointer)
func sigprocmask(sig int32, new, old unsafe.Pointer) int32
func getrlimit(kind int32, limit unsafe.Pointer) int32
func netpollinit()
func netpollopen(fd uintptr, pd *pollDesc) int32
func netpollclose(fd uintptr) int32
func netpollarm(pd *pollDesc, mode int)

func os_sigpipe() {
	gothrow("too many writes on closed pipe")
}

func signame(int32) *byte
func sigpanic() {
	g := getg()
	if !canpanic(g) {
		gothrow("unexpected signal during runtime execution")
	}

	switch g.sig {
	case _SIGBUS:
		if g.sigcode0 == _BUS_ADRERR && g.sigcode1 < 0x1000 || g.paniconfault {
			panicmem()
		}
		print("unexpected fault address ", hex(g.sigcode1), "\n")
		gothrow("fault")
	case _SIGSEGV:
		if (g.sigcode0 == 0 || g.sigcode0 == _SEGV_MAPERR || g.sigcode0 == _SEGV_ACCERR) && g.sigcode1 < 0x1000 || g.paniconfault {
			panicmem()
		}
		print("unexpected fault address ", hex(g.sigcode1), "\n")
		gothrow("fault")
	case _SIGFPE:
		switch g.sigcode0 {
		case _FPE_INTDIV:
			panicdivide()
		case _FPE_INTOVF:
			panicoverflow()
		}
		panicfloat()
	}
	panic(errorString(gostringnocopy(signame(g.sig))))
}
