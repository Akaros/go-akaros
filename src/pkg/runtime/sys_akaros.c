// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//
// System calls and other sys stuff for Akaros
//

#include "runtime.h"
#include "cgocall.h"
#include "defs_GOOS_GOARCH.h"
#include "os_GOOS.h"
#include "zsyscall_akaros.h"
#include "../../cmd/ld/textflag.h"

// We extern these libc functions here, so we don't have to reimplement the logic
// they entails just for compilation in the kenc world.
// We use asmcgocall() to call them.
#pragma cgo_import_static gcc_syscall
#pragma cgo_import_static gcc_futex
#pragma cgo_import_static gcc_myield
#pragma cgo_import_static gcc_sigprocmask
#pragma cgo_import_static gcc_enable_profalarm
#pragma cgo_import_static gcc_disable_profalarm
typedef void (*gcc_call_t)(void *arg);
extern gcc_call_t gcc_syscall;
extern gcc_call_t gcc_futex;
extern gcc_call_t gcc_myield;
extern gcc_call_t gcc_sigprocmask;
extern gcc_call_t gcc_enable_profalarm;
extern gcc_call_t gcc_disable_profalarm;

#pragma textflag NOSPLIT
static intgo strlen(int8 *string)
{
	int8 *temp = string;
	while(*temp != '\0')
		temp++;
	return temp-string;
}

#pragma textflag NOSPLIT
static inline bool mult_will_overflow_int64(int64 a, int64 b)
{
    if (!a)
        return false;
    return ((uint64)(-1) >> 1) / a < b;
}

#pragma textflag NOSPLIT
static inline int64 tsc2sec(int64 tsc_time)
{
    return tsc_time / __procinfo.tsc_freq;
}

#pragma textflag NOSPLIT
static inline int64 tsc2msec(int64 tsc_time)
{
    if (mult_will_overflow_int64(tsc_time, 1000LL))
        return tsc2sec(tsc_time) * 1000LL;
    else
        return (tsc_time * 1000LL) / __procinfo.tsc_freq;
}

#pragma textflag NOSPLIT
static inline int64 tsc2usec(int64 tsc_time)
{
    if (mult_will_overflow_int64(tsc_time, 1000000LL))
        return tsc2msec(tsc_time) * 1000LL;
    else
        return (tsc_time * 1000000LL) / __procinfo.tsc_freq;
}

#pragma textflag NOSPLIT
static inline int64 tsc2nsec(int64 tsc_time)
{
    if (mult_will_overflow_int64(tsc_time, 1000000000LL))
        return tsc2usec(tsc_time) * 1000LL;
    else
        return (tsc_time * 1000000000LL) / __procinfo.tsc_freq;
}

#pragma textflag NOSPLIT
static inline int64 sec2tsc(int64 sec)
{
    if (mult_will_overflow_int64(sec, __procinfo.tsc_freq))
        return (int64)(-1);
    else
        return sec * __procinfo.tsc_freq;
}

#pragma textflag NOSPLIT
static inline int64 msec2tsc(int64 msec)
{
    if (mult_will_overflow_int64(msec, __procinfo.tsc_freq))
        return sec2tsc(msec / 1000LL);
    else
        return (msec * __procinfo.tsc_freq) / 1000LL;
}

#pragma textflag NOSPLIT
static inline int64 usec2tsc(int64 usec)
{
    if (mult_will_overflow_int64(usec, __procinfo.tsc_freq))
        return msec2tsc(usec / 1000LL);
    else
        return (usec * __procinfo.tsc_freq) / 1000000LL;
}

#pragma textflag NOSPLIT
static inline int64 nsec2tsc(int64 nsec)
{
    if (mult_will_overflow_int64(nsec, __procinfo.tsc_freq))
        return usec2tsc(nsec / 1000LL);
    else
        return (nsec * __procinfo.tsc_freq) / 1000000000LL;
}

// Wrapper for making an akaros syscall through gcc
#define __akaros_syscall(sysc, n, a0, a1, a2, a3, a4, a5, perrno) \
do { \
	sysc->num = n;       \
	sysc->err = 0;       \
	sysc->retval = 0;    \
	sysc->flags = 0;     \
	sysc->ev_q = 0;      \
	sysc->u_data = 0;    \
	sysc->arg0 = a0;     \
	sysc->arg1 = a1;     \
	sysc->arg2 = a2;     \
	sysc->arg3 = a3;     \
	sysc->arg4 = a4;     \
	sysc->arg5 = a5;     \
	sysc->errstr[0] = 0; \
	runtime·asmcgocall(gcc_syscall, sysc); \
	if(perrno != nil) \
    	*perrno = sysc->err; \
} while(0);

#define akaros_syscall(sysc, n, a0, a1, a2, a3, a4, a5, perrno) \
	__akaros_syscall(((sysc)), ((uint32)(n)),                   \
	                 ((intgo)(a0)), ((intgo)(a1)),              \
	                 ((intgo)(a2)), ((intgo)(a3)),              \
	                 ((intgo)(a4)), ((intgo)(a5)),              \
	                 ((int32*)(perrno)))

#pragma textflag NOSPLIT
int32 runtime·getpid(void)
{
	return 	__procinfo.pid;
}

#pragma textflag NOSPLIT
int32 runtime·read(int32 fd, void* buf, int32 count)
{
	int32 errno;
	SyscallArg *sysc = (SyscallArg *)(g->sysc);
	akaros_syscall(sysc, SYS_read, fd, buf, count, 0, 0, 0, &errno);
	return sysc->retval;
}

#pragma textflag NOSPLIT
int32 runtime·write(uintptr fd, void* buf, int32 count)
{
	int32 errno;
	SyscallArg *sysc = (SyscallArg *)(g->sysc);
	akaros_syscall(sysc, SYS_write, fd, buf, count, 0, 0, 0, &errno);
	return sysc->retval;
}

#pragma textflag NOSPLIT
int32 runtime·open(int8* pathname, int32 flags, int32 mode)
{
	int32 errno;
	intgo len = strlen(pathname);
	SyscallArg *sysc = (SyscallArg *)(g->sysc);
	akaros_syscall(sysc, SYS_openat, AT_FDCWD, pathname, len, flags, mode, 0, &errno);
	return sysc->retval;
}

#pragma textflag NOSPLIT
int32 runtime·close(int32 fd)
{
	int32 errno;
	SyscallArg *sysc = (SyscallArg *)(g->sysc);
	akaros_syscall(sysc, SYS_close, fd, 0, 0, 0, 0, 0, &errno);
	return sysc->retval;
}

#pragma textflag NOSPLIT
uint8* runtime·mmap(byte* addr, uintptr len, int32 prot,
                    int32 flags, int32 fd, uint32 offset)
{
	int32 errno;
	SyscallArg *sysc = (SyscallArg *)(g->sysc);
	akaros_syscall(sysc, SYS_mmap, addr, len, prot, flags, fd, offset, &errno);
	return (uint8*)sysc->retval;
}

#pragma textflag NOSPLIT
void runtime·munmap(byte* addr, uintptr len)
{
	SyscallArg *sysc = (SyscallArg *)(g->sysc);
	akaros_syscall(sysc, SYS_munmap, addr, len, 0, 0, 0, 0, nil);
}

#pragma textflag NOSPLIT
void runtime·osyield(void)
{
	runtime·asmcgocall(gcc_myield, nil);
}

#pragma textflag NOSPLIT
void runtime·usleep(uint32 usec)
{
	SyscallArg *sysc = (SyscallArg *)(g->sysc);
	akaros_syscall(sysc, SYS_block, usec, 0, 0, 0, 0, 0, nil);
}

#pragma textflag NOSPLIT
int64 runtime·nanotime(void)
{
	return tsc2nsec(runtime·cputicks());
}

#pragma textflag NOSPLIT
void time·now(int64 sec, int32 nsec)
{
	// NOTE: we should add real time base to the nsec since start,
	// but gettimeofday syscall is missing on akaros.
	int64 time = tsc2nsec(runtime·cputicks() - __proc_global_info.tsc_cycles_last) + __proc_global_info.walltime_ns_last;
	sec = time / 1000000000LL;
	nsec = time - sec * 1000000000LL;
	FLUSH(&sec);
	FLUSH(&nsec);
}

#pragma textflag NOSPLIT
void runtime·exit(int32 status)
{
	intgo pid = runtime·getpid();
	SyscallArg *sysc = (SyscallArg *)(g->sysc);
	akaros_syscall(sysc, SYS_proc_destroy, pid, status, 0, 0, 0, 0, nil);
	runtime·throw("Exit Returned: We should never get here!");
}

#pragma textflag NOSPLIT
int32 runtime·futex(uint32 *uaddr, int32 op, uint32 val,
                    Timespec *timeout, uint32 *uaddr2, uint32 val3)
{
	FutexArg a;
	a.uaddr = (int32*)uaddr;
	a.op = op;
	a.val = val;
	a.timeout = timeout;
	a.uaddr2 = (int32*)uaddr2;
	a.val3 = val3;
	runtime·asmcgocall(gcc_futex, &a);
	return a.retval;
}

#pragma textflag NOSPLIT
void runtime·raise(int32 sig)
{
	USED(sig);
}

#pragma textflag NOSPLIT
int32 runtime·sigprocmask(int32 how, Sigset *set, Sigset *oldset)
{
	SigprocmaskArg a;
	a.how = how;
	a.set = set;
	a.oset = oldset;
	runtime·asmcgocall(gcc_sigprocmask, &a);
	return a.retval;
}

#pragma textflag NOSPLIT
void runtime·enable_profalarm(uint64 usecs)
{
	runtime·asmcgocall(gcc_enable_profalarm, &usecs);
}

#pragma textflag NOSPLIT
void runtime·disable_profalarm()
{
	runtime·asmcgocall(gcc_disable_profalarm, nil);
}

//TEXT runtime·exit1(SB),7,$0
//	MOVL	$1, AX	// exit - exit the current os thread
//	MOVL	4(SP), BX
//	CALL	*runtime·_vdso(SB)
//	INT $3	// not reached
//	RET
//
//TEXT runtime·getrlimit(SB),7,$0
//	MOVL	$191, AX		// syscall - ugetrlimit
//	MOVL	4(SP), BX
//	MOVL	8(SP), CX
//	CALL	*runtime·_vdso(SB)
//	RET
//
//TEXT runtime·mincore(SB),7,$0-24
//	MOVL	$218, AX			// syscall - mincore
//	MOVL	4(SP), BX
//	MOVL	8(SP), CX
//	MOVL	12(SP), DX
//	CALL	*runtime·_vdso(SB)
//	RET
//
//TEXT runtime·rtsigprocmask(SB),7,$0
//	MOVL	$175, AX		// syscall entry
//	MOVL	4(SP), BX
//	MOVL	8(SP), CX
//	MOVL	12(SP), DX
//	MOVL	16(SP), SI
//	CALL	*runtime·_vdso(SB)
//	CMPL	AX, $0xfffff001
//	JLS	2(PC)
//	INT $3
//	RET
//
//TEXT runtime·rt_sigaction(SB),7,$0
//	MOVL	$174, AX		// syscall - rt_sigaction
//	MOVL	4(SP), BX
//	MOVL	8(SP), CX
//	MOVL	12(SP), DX
//	MOVL	16(SP), SI
//	CALL	*runtime·_vdso(SB)
//	RET
//
//TEXT runtime·madvise(SB),7,$0
//	MOVL	$219, AX	// madvise
//	MOVL	4(SP), BX
//	MOVL	8(SP), CX
//	MOVL	12(SP), DX
//	CALL	*runtime·_vdso(SB)
//	// ignore failure - maybe pages are locked
//	RET
//
//// int32 clone(int32 flags, void *stack, M *mp, G *gp, void (*fn)(void));
//TEXT runtime·clone(SB),7,$0
//	MOVL	$120, AX	// clone
//	MOVL	flags+4(SP), BX
//	MOVL	stack+8(SP), CX
//	MOVL	$0, DX	// parent tid ptr
//	MOVL	$0, DI	// child tid ptr
//
//	// Copy mp, gp, fn off parent stack for use by child.
//	SUBL	$16, CX
//	MOVL	mm+12(SP), SI
//	MOVL	SI, 0(CX)
//	MOVL	gg+16(SP), SI
//	MOVL	SI, 4(CX)
//	MOVL	fn+20(SP), SI
//	MOVL	SI, 8(CX)
//	MOVL	$1234, 12(CX)
//
//	// cannot use CALL *runtime·_vdso(SB) here, because
//	// the stack changes during the system call (after
//	// CALL *runtime·_vdso(SB), the child is still using
//	// the parent's stack when executing its RET instruction).
//	INT	$0x80
//
//	// In parent, return.
//	CMPL	AX, $0
//	JEQ	2(PC)
//	RET
//
//	// Paranoia: check that SP is as we expect.
//	MOVL	12(SP), BP
//	CMPL	BP, $1234
//	JEQ	2(PC)
//	INT	$3
//
//	// Initialize AX to Linux tid
//	MOVL	$224, AX
//	CALL	*runtime·_vdso(SB)
//
//	// In child on new stack.  Reload registers (paranoia).
//	MOVL	0(SP), BX	// m
//	MOVL	4(SP), DX	// g
//	MOVL	8(SP), SI	// fn
//
//	MOVL	AX, m_procid(BX)	// save tid as m->procid
//
//	// set up ldt 7+id to point at m->tls.
//	// newosproc left the id in tls[0].
//	LEAL	m_tls(BX), BP
//	MOVL	0(BP), DI
//	ADDL	$7, DI	// m0 is LDT#7. count up.
//	// setldt(tls#, &tls, sizeof tls)
//	PUSHAL	// save registers
//	PUSHL	$32	// sizeof tls
//	PUSHL	BP	// &tls
//	PUSHL	DI	// tls #
//	CALL	runtime·setldt(SB)
//	POPL	AX
//	POPL	AX
//	POPL	AX
//	POPAL
//
//	// Now segment is established.  Initialize m, g.
//	get_tls(AX)
//	MOVL	DX, g(AX)
//	MOVL	BX, m(AX)
//
//	CALL	runtime·stackcheck(SB)	// smashes AX, CX
//	MOVL	0(DX), DX	// paranoia; check they are not nil
//	MOVL	0(BX), BX
//
//	// more paranoia; check that stack splitting code works
//	PUSHAL
//	CALL	runtime·emptyfunc(SB)
//	POPAL
//
//	CALL	SI	// fn()
//	CALL	runtime·exit1(SB)
//	MOVL	$0x1234, 0x1005
//	RET
//
//TEXT runtime·sigaltstack(SB),7,$-8
//	MOVL	$186, AX	// sigaltstack
//	MOVL	new+4(SP), BX
//	MOVL	old+8(SP), CX
//	CALL	*runtime·_vdso(SB)
//	CMPL	AX, $0xfffff001
//	JLS	2(PC)
//	INT	$3
//	RET
//
//// <asm-i386/ldt.h>
//// struct user_desc {
////	unsigned int  entry_number;
////	unsigned long base_addr;
////	unsigned int  limit;
////	unsigned int  seg_32bit:1;
////	unsigned int  contents:2;
////	unsigned int  read_exec_only:1;
////	unsigned int  limit_in_pages:1;
////	unsigned int  seg_not_present:1;
////	unsigned int  useable:1;
//// };
//#define SEG_32BIT 0x01
//// contents are the 2 bits 0x02 and 0x04.
//#define CONTENTS_DATA 0x00
//#define CONTENTS_STACK 0x02
//#define CONTENTS_CODE 0x04
//#define READ_EXEC_ONLY 0x08
//#define LIMIT_IN_PAGES 0x10
//#define SEG_NOT_PRESENT 0x20
//#define USEABLE 0x40
//
//TEXT runtime·sched_getaffinity(SB),7,$0
//	MOVL	$242, AX		// syscall - sched_getaffinity
//	MOVL	4(SP), BX
//	MOVL	8(SP), CX
//	MOVL	12(SP), DX
//	CALL	*runtime·_vdso(SB)
//	RET
//
int32 runtime·epollcreate(int32 size)
{ USED(size); return -1; }
//TEXT runtime·epollcreate(SB),7,$0
//	MOVL    $254, AX
//	MOVL	4(SP), BX
//	CALL	*runtime·_vdso(SB)
//	RET
//
int32 runtime·epollcreate1(int32 flags)
{ USED(flags); return -1; }
//TEXT runtime·epollcreate1(SB),7,$0
//	MOVL    $329, AX
//	MOVL	4(SP), BX
//	CALL	*runtime·_vdso(SB)
//	RET
//
int32 runtime·epollctl(int32 epfd, int32 op, int32 fd, EpollEvent *ev)
{ USED(epfd); USED(op); USED(fd); USED(ev); return -1; }
//TEXT runtime·epollctl(SB),7,$0
//	MOVL	$255, AX
//	MOVL	4(SP), BX
//	MOVL	8(SP), CX
//	MOVL	12(SP), DX
//	MOVL	16(SP), SI
//	CALL	*runtime·_vdso(SB)
//	RET
//
int32 runtime·epollwait(int32 epfd, EpollEvent *ev, int32 nev, int32 timeout)
{ USED(epfd); USED(ev); USED(nev); USED(timeout); return -1; }
//TEXT runtime·epollwait(SB),7,$0
//	MOVL	$256, AX
//	MOVL	4(SP), BX
//	MOVL	8(SP), CX
//	MOVL	12(SP), DX
//	MOVL	16(SP), SI
//	CALL	*runtime·_vdso(SB)
//	RET
//
void runtime·closeonexec(int32 fd)
{ USED(fd); }
//TEXT runtime·closeonexec(SB),7,$0
//	MOVL	$55, AX  // fcntl
//	MOVL	4(SP), BX  // fd
//	MOVL	$2, CX  // F_SETFD
//	MOVL	$1, DX  // FD_CLOEXEC
//	CALL	*runtime·_vdso(SB)
//	RET
