// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

/*
Input to cgo -cdefs

go-akaros-386 tool cgo -cdefs defs_akaros.go > defs_akaros_386.h
go-akaros-amd64 tool cgo -cdefs defs_akaros.go > defs_akaros_amd64.h
*/

package parlib

/*
#include <futex.h>
#include <signal.h>
#include <stdint.h>
#include <parlib/alarm.h>
#include <parlib/mcs.h>
#include <parlib/parlib.h>
#include <parlib/uthread.h>
#include <parlib/vcore.h>
#include <parlib/serialize.h>
#include <ros/errno.h>
#include <ros/fs.h>
#include <ros/memlayout.h>
#include <ros/mman.h>
#include <ros/procinfo.h>
#include <ros/trapframe.h>
#include "parlib/gcc_akaros.h"
*/
import "C"

const (
	PROC_DUP_FGRP = C.PROC_DUP_FGRP

	FUTEX_WAIT = C.FUTEX_WAIT
	FUTEX_WAKE = C.FUTEX_WAKE

	EINTR  = C.EINTR
	EAGAIN = C.EAGAIN
	ENOMEM = C.ENOMEM

	PROT_NONE  = C.PROT_NONE
	PROT_READ  = C.PROT_READ
	PROT_WRITE = C.PROT_WRITE
	PROT_EXEC  = C.PROT_EXEC

	MAP_ANON     = C.MAP_ANONYMOUS
	MAP_PRIVATE  = C.MAP_PRIVATE
	MAP_FIXED    = C.MAP_FIXED
	MAP_POPULATE = C.MAP_POPULATE

	SA_RESTART  = C.SA_RESTART
	SA_ONSTACK  = C.SA_ONSTACK
	SA_SIGINFO  = C.SA_SIGINFO

	SIGHUP    = C.SIGHUP
	SIGINT    = C.SIGINT
	SIGQUIT   = C.SIGQUIT
	SIGILL    = C.SIGILL
	SIGTRAP   = C.SIGTRAP
	SIGABRT   = C.SIGABRT
	SIGBUS    = C.SIGBUS
	SIGFPE    = C.SIGFPE
	SIGKILL   = C.SIGKILL
	SIGUSR1   = C.SIGUSR1
	SIGSEGV   = C.SIGSEGV
	SIGUSR2   = C.SIGUSR2
	SIGPIPE   = C.SIGPIPE
	SIGALRM   = C.SIGALRM
	SIGSTKFLT = C.SIGSTKFLT
	SIGCHLD   = C.SIGCHLD
	SIGCONT   = C.SIGCONT
	SIGSTOP   = C.SIGSTOP
	SIGTSTP   = C.SIGTSTP
	SIGTTIN   = C.SIGTTIN
	SIGTTOU   = C.SIGTTOU
	SIGURG    = C.SIGURG
	SIGXCPU   = C.SIGXCPU
	SIGXFSZ   = C.SIGXFSZ
	SIGVTALRM = C.SIGVTALRM
	SIGPROF   = C.SIGPROF
	SIGWINCH  = C.SIGWINCH
	SIGIO     = C.SIGIO
	SIGPWR    = C.SIGPWR
	SIGSYS    = C.SIGSYS
	NSIG      = C._NSIG

	SIGRTMIN = C.__SIGRTMIN
	SIGRTMAX = C.__SIGRTMAX

	FPE_INTDIV = C.FPE_INTDIV
	FPE_INTOVF = C.FPE_INTOVF
	FPE_FLTDIV = C.FPE_FLTDIV
	FPE_FLTOVF = C.FPE_FLTOVF
	FPE_FLTUND = C.FPE_FLTUND
	FPE_FLTRES = C.FPE_FLTRES
	FPE_FLTINV = C.FPE_FLTINV
	FPE_FLTSUB = C.FPE_FLTSUB

	BUS_ADRALN = C.BUS_ADRALN
	BUS_ADRERR = C.BUS_ADRERR
	BUS_OBJERR = C.BUS_OBJERR

	SI_USER     = C.SI_USER
	SEGV_MAPERR = C.SEGV_MAPERR
	SEGV_ACCERR = C.SEGV_ACCERR

	AT_FDCWD = C.AT_FDCWD
)

type Sigset C.sigset_t
type Vcore C.struct_vcore
type Pcore C.struct_pcore
type ProcinfoType C.procinfo_t
type GlobalProcinfoType C.struct_proc_global_info
type Ucq C.struct_ucq
type EventQueue C.struct_event_queue
type EventMbox C.struct_event_mbox
type Timespec C.struct_timespec
type Timeval C.struct_timeval
type Itimerval C.struct_itimerval
type Sigaction C.struct_parlib_sigaction
type Siginfo C.siginfo_t
type HwTrapframe C.struct_hw_trapframe
type SwTrapframe C.struct_sw_trapframe
type UserContext C.struct_user_context
type AlarmWaiterTailQEntry C.parlib_alarm_waiter_tailq_entry_t
type AlarmWaiter C.struct_parlib_alarm_waiter

type SyscallArg C.gcc_syscall_arg_t
type FutexArg C.gcc_futex_arg_t
type SigactionArg C.gcc_sigaction_arg_t
type SigprocmaskArg C.gcc_sigprocmask_arg_t

type SerializedData struct {
	Len C.size_t
	Buf [1]byte
}
