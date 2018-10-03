// Created by cgo -cdefs - DO NOT EDIT
// cgo -cdefs defs_akaros.go defsbogus_akaros.go


enum {
	PROC_DUP_FGRP	= 0x1,

	FUTEX_WAIT	= 0x0,
	FUTEX_WAKE	= 0x1,

	EINTR	= 0x4,
	EAGAIN	= 0xb,
	ENOMEM	= 0xc,

	PROT_NONE	= 0x0,
	PROT_READ	= 0x1,
	PROT_WRITE	= 0x2,
	PROT_EXEC	= 0x4,

	MAP_ANON	= 0x20,
	MAP_PRIVATE	= 0x2,
	MAP_FIXED	= 0x10,
	MAP_POPULATE	= 0x8000,

	SA_RESTART	= 0x10000000,
	SA_ONSTACK	= 0x8000000,
	SA_SIGINFO	= 0x4,

	SIGHUP		= 0x1,
	SIGINT		= 0x2,
	SIGQUIT		= 0x3,
	SIGILL		= 0x4,
	SIGTRAP		= 0x5,
	SIGABRT		= 0x6,
	SIGBUS		= 0x7,
	SIGFPE		= 0x8,
	SIGKILL		= 0x9,
	SIGUSR1		= 0xa,
	SIGSEGV		= 0xb,
	SIGUSR2		= 0xc,
	SIGPIPE		= 0xd,
	SIGALRM		= 0xe,
	SIGSTKFLT	= 0x10,
	SIGCHLD		= 0x11,
	SIGCONT		= 0x12,
	SIGSTOP		= 0x13,
	SIGTSTP		= 0x14,
	SIGTTIN		= 0x15,
	SIGTTOU		= 0x16,
	SIGURG		= 0x17,
	SIGXCPU		= 0x18,
	SIGXFSZ		= 0x19,
	SIGVTALRM	= 0x1a,
	SIGPROF		= 0x1b,
	SIGWINCH	= 0x1c,
	SIGIO		= 0x1d,
	SIGPWR		= 0x1e,
	SIGSYS		= 0x1f,

	SIGRTMIN	= 0x20,
	SIGRTMAX	= 0x29,

	FPE_INTDIV	= 0x1,
	FPE_INTOVF	= 0x2,
	FPE_FLTDIV	= 0x3,
	FPE_FLTOVF	= 0x4,
	FPE_FLTUND	= 0x5,
	FPE_FLTRES	= 0x6,
	FPE_FLTINV	= 0x7,
	FPE_FLTSUB	= 0x8,

	BUS_ADRALN	= 0x1,
	BUS_ADRERR	= 0x2,
	BUS_OBJERR	= 0x3,

	SI_USER		= 0x0,
	SEGV_MAPERR	= 0x1,
	SEGV_ACCERR	= 0x2,

	AT_FDCWD	= -0x64,
};

typedef struct Vcore Vcore;
typedef struct Pcore Pcore;
typedef struct ProcinfoType ProcinfoType;
typedef struct GlobalProcinfoType GlobalProcinfoType;
typedef struct Ucq Ucq;
typedef struct EventQueue EventQueue;
typedef struct EventMbox EventMbox;
typedef struct Timespec Timespec;
typedef struct Timeval Timeval;
typedef struct Itimerval Itimerval;
typedef struct SigactionT SigactionT;
typedef struct StackT StackT;
typedef struct Siginfo Siginfo;
typedef struct HwTrapframe HwTrapframe;
typedef struct SwTrapframe SwTrapframe;
typedef struct UserContext UserContext;
typedef struct AlarmWaiterTailQEntry AlarmWaiterTailQEntry;
typedef struct AlarmWaiter AlarmWaiter;
typedef struct SyscallArg SyscallArg;
typedef struct FutexArg FutexArg;
typedef struct SigactionArg SigactionArg;
typedef struct SigprocmaskArg SigprocmaskArg;

#pragma pack on

typedef	uint64	Sigset;
struct Vcore {
	byte	*dummy_ptr1;
	byte	*dummy_ptr2;
	uint32	pcoreid;
	bool	valid;
	byte	Pad_cgo_0[3];
	uint32	nr_preempts_sent;
	uint32	nr_preempts_done;
	uint64	preempt_pending;
	uint64	resume_ticks;
	uint64	total_ticks;
};
struct Pcore {
	uint32	vcoreid;
	bool	valid;
	byte	Pad_cgo_0[3];
};
struct ProcinfoType {
	int32	pid;
	int32	ppid;
	uint64	max_vcores;
	uint64	tsc_freq;
	uint64	timing_overhead;
	uint64	program_end;
	bool	is_mcp;
	byte	Pad_cgo_0[7];
	uint64	res_grant[3];
	Vcore	vcoremap[255];
	uint32	num_vcores;
	Pcore	pcoremap[255];
	uint32	coremap_seqctr;
};
struct GlobalProcinfoType {
	uint64	cpu_feats[2];
	uint64	x86_default_xcr0;
	uint64	tsc_freq;
	uint64	tsc_overhead;
	uint64	bus_freq;
	uint64	walltime_ns_last;
	uint64	tsc_cycles_last;
	byte	Pad_cgo_0[4032];
};
struct Ucq {
	byte	*prod_idx;
	byte	*spare_pg;
	byte	*nr_extra_pgs;
	byte	*cons_idx;
	bool	prod_overflow;
	bool	ucq_ready;
	byte	Pad_cgo_0[2];
	uint32	u_lock[2];
	byte	Pad_cgo_1[4];
};
struct EventQueue {
	EventMbox	*ev_mbox;
	int32	ev_flags;
	bool	ev_alert_pending;
	byte	Pad_cgo_0[3];
	uint32	ev_vcore;
	byte	Pad_cgo_1[4];
	void	*ev_handler;
	byte	*ev_udata;
};
struct EventMbox {
	int32	type;
	byte	Pad_cgo_0[4];
	Ucq	ucq;
	byte	Pad_cgo_1[24];
};
struct Timespec {
	int64	tv_sec;
	int64	tv_nsec;
};
struct Timeval {
	int64	tv_sec;
	int64	tv_usec;
};
struct Itimerval {
	Timeval	it_interval;
	Timeval	it_value;
};
struct SigactionT {
	void	*sa_sigact;
	uint64	sa_mask;
	uint32	sa_flags;
	uint32	padding;
	void	*sa_restorer;
};
struct StackT {
	byte	*ss_sp;
	uint64	ss_size;
	int32	ss_flags;
	byte	Pad_cgo_0[4];
};
struct Siginfo {
	int32	si_signo;
	int32	si_errno;
	int32	si_code;
	int32	si_pid;
	uint32	si_uid;
	byte	Pad_cgo_0[4];
	byte	*si_addr;
	int32	si_status;
	byte	Pad_cgo_1[4];
	int64	si_band;
	byte	si_value[8];
};
struct HwTrapframe {
	uint64	tf_gsbase;
	uint64	tf_fsbase;
	uint64	tf_rax;
	uint64	tf_rbx;
	uint64	tf_rcx;
	uint64	tf_rdx;
	uint64	tf_rbp;
	uint64	tf_rsi;
	uint64	tf_rdi;
	uint64	tf_r8;
	uint64	tf_r9;
	uint64	tf_r10;
	uint64	tf_r11;
	uint64	tf_r12;
	uint64	tf_r13;
	uint64	tf_r14;
	uint64	tf_r15;
	uint32	tf_trapno;
	uint32	tf_padding5;
	uint32	tf_err;
	uint32	tf_padding4;
	uint64	tf_rip;
	uint16	tf_cs;
	uint16	tf_padding3;
	uint32	tf_padding2;
	uint64	tf_rflags;
	uint64	tf_rsp;
	uint16	tf_ss;
	uint16	tf_padding1;
	uint32	tf_padding0;
};
struct SwTrapframe {
	uint64	tf_gsbase;
	uint64	tf_fsbase;
	uint64	tf_rbx;
	uint64	tf_rbp;
	uint64	tf_r12;
	uint64	tf_r13;
	uint64	tf_r14;
	uint64	tf_r15;
	uint64	tf_rip;
	uint64	tf_rsp;
	uint32	tf_mxcsr;
	uint16	tf_fpucw;
	uint16	tf_padding0;
};
struct UserContext {
	int32	type;
	byte	Pad_cgo_0[4];
	byte	tf[208];
};
struct AlarmWaiterTailQEntry {
	AlarmWaiter	*tqe_next;
	AlarmWaiter	**tqe_prev;
};
struct AlarmWaiter {
	uint64	wake_up_time;
	void	*func;
	byte	*data;
	AlarmWaiterTailQEntry	next;
	bool	on_tchain;
	byte	Pad_cgo_0[7];
};

struct SyscallArg {
	uint32	num;
	int32	err;
	int64	retval;
	byte	*flags;
	EventQueue	*ev_q;
	byte	*u_data;
	int64	arg0;
	int64	arg1;
	int64	arg2;
	int64	arg3;
	int64	arg4;
	int64	arg5;
	int8	errstr[128];
};
struct FutexArg {
	int32	*uaddr;
	int32	op;
	int32	val;
	Timespec	*timeout;
	int32	*uaddr2;
	int32	val3;
	int32	retval;
};
struct SigactionArg {
	int32	sig;
	byte	Pad_cgo_0[4];
	SigactionT	*act;
	SigactionT	*oact;
	int32	ret;
	byte	Pad_cgo_1[4];
};
struct SigprocmaskArg {
	int32	how;
	byte	Pad_cgo_0[4];
	uint64	*set;
	uint64	*oset;
	int32	retval;
	byte	Pad_cgo_1[4];
};


#pragma pack off
// Created by cgo -cdefs - DO NOT EDIT
// cgo -cdefs defs_akaros.go defsbogus_akaros.go


enum {
	ITIMER_REAL	= 0,
	ITIMER_VIRTUAL	= 0,
	ITIMER_PROF	= 0,

	EPOLLIN		= 0,
	EPOLLOUT	= 0,
	EPOLLERR	= 0,
	EPOLLHUP	= 0,
	EPOLLRDHUP	= 0,
	EPOLLET		= 0,
	EPOLL_CLOEXEC	= 0,
	EPOLL_CTL_ADD	= 0,
	EPOLL_CTL_DEL	= 0,
	EPOLL_CTL_MOD	= 0,
};


#pragma pack on

typedef	void	*EpollEvent;


#pragma pack off
