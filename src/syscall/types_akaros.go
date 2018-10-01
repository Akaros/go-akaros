// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

/*
Input to cgo -godefs.  See also mkerrors.sh and mkall.sh
*/

// +godefs map struct_in_addr [4]byte /* in_addr */
// +godefs map struct_in6_addr [16]byte /* in6_addr */

package syscall

/*
#define _LARGEFILE_SOURCE
#define _LARGEFILE64_SOURCE
#define _FILE_OFFSET_BITS 64
#define _GNU_SOURCE

#include <dirent.h>
#include <unistd.h>
#include <fcntl.h>
#include <termios.h>
#include <net/if.h>
#include <netinet/in.h>
#include <netinet/icmp6.h>
#include <sys/un.h>
#include <sys/mman.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/time.h>
#include <sys/socket.h>
#include <sys/resource.h>
#include <bits/sockaddr.h>
#include <ros/glibc-asm/ioctls.h>
#include <ros/event.h>
#include <ros/syscall.h>

#define BIT8SZ      1
#define BIT16SZ     2
#define BIT32SZ     4
#define BIT64SZ     8
#define QIDSZ   (BIT8SZ+BIT32SZ+BIT64SZ)
#define STATFIXLEN  (BIT16SZ+QIDSZ+5*BIT16SZ+4*BIT32SZ+1*BIT64SZ)

enum {
	sizeofPtr = sizeof(void*),
};

union sockaddr_all {
	struct sockaddr s1;	// this one gets used for fields
	struct sockaddr_in s2;	// these pad it out
	struct sockaddr_in6 s3;
	struct sockaddr_un s4;
};

struct sockaddr_any {
	struct sockaddr addr;
	char pad[sizeof(union sockaddr_all) - sizeof(struct sockaddr)];
};

struct my_sockaddr_un {
	sa_family_t sun_family;
	char sun_path[108];
};
*/
import "C"

// Machine characteristics; for internal use.

const (
	sizeofPtr      = C.sizeofPtr
	sizeofShort    = C.sizeof_short
	sizeofInt      = C.sizeof_int
	sizeofLong     = C.sizeof_long
	sizeofLongLong = C.sizeof_longlong
	PathMax        = C.PATH_MAX
)

// Basic types

type (
	_C_short     C.short
	_C_int       C.int
	_C_long      C.long
	_C_long_long C.longlong
)

type EventMsg C.struct_event_msg

// Time

type Timespec C.struct_timespec

type Timeval C.struct_timeval

type Time_t C.time_t

type Tms C.struct_tms

type Utimbuf C.struct_utimbuf

// Processes

type Rusage C.struct_rusage

type Rlimit C.struct_rlimit

type _Gid_t C.gid_t

// Files

type Stat_t C.struct_stat

type Statfs_t C.struct_statfs

type Dirent C.struct_dirent

type Fsid C.fsid_t

const (
	SEEK_SET = C.SEEK_SET
	SEEK_CUR = C.SEEK_CUR
	SEEK_END = C.SEEK_END
)

// Syscall interface stuff

type Childfdmap_t C.struct_childfdmap

// 9p stuff

const (
	BIT8SZ     = C.BIT8SZ
	BIT16SZ    = C.BIT16SZ
	BIT32SZ    = C.BIT32SZ
	BIT64SZ    = C.BIT64SZ
	QIDSZ      = C.QIDSZ
	STATFIXLEN = C.STATFIXLEN
)

const (
	WSTAT_MODE   = C.WSTAT_MODE
	WSTAT_ATIME  = C.WSTAT_ATIME
	WSTAT_MTIME  = C.WSTAT_MTIME
	WSTAT_LENGTH = C.WSTAT_LENGTH
	WSTAT_NAME   = C.WSTAT_NAME
	WSTAT_UID    = C.WSTAT_UID
	WSTAT_GID    = C.WSTAT_GID
	WSTAT_MUID   = C.WSTAT_MUID
)

// Sockets

type RawSockaddrInet4 C.struct_sockaddr_in

type RawSockaddrInet6 C.struct_sockaddr_in6

type RawSockaddrUnix C.struct_my_sockaddr_un

type RawSockaddr C.struct_sockaddr

type RawSockaddrAny C.struct_sockaddr_any

type _Socklen C.socklen_t

type Linger C.struct_linger

type Iovec C.struct_iovec

type IPMreq C.struct_ip_mreq

type IPv6Mreq C.struct_ipv6_mreq

type Msghdr C.struct_msghdr

type Cmsghdr C.struct_cmsghdr

type Inet6Pktinfo C.struct_in6_pktinfo

type IPv6MTUInfo C.struct_ip6_mtuinfo

type ICMPv6Filter C.struct_icmp6_filter

type Ucred C.struct_ucred

const (
	SizeofSockaddrInet4 = C.sizeof_struct_sockaddr_in
	SizeofSockaddrInet6 = C.sizeof_struct_sockaddr_in6
	SizeofSockaddrUnix  = C.sizeof_struct_my_sockaddr_un
	SizeofSockaddrAny   = C.sizeof_struct_sockaddr_any
	SizeofLinger        = C.sizeof_struct_linger
	SizeofIPMreq        = C.sizeof_struct_ip_mreq
	SizeofIPv6Mreq      = C.sizeof_struct_ipv6_mreq
	SizeofMsghdr        = C.sizeof_struct_msghdr
	SizeofCmsghdr       = C.sizeof_struct_cmsghdr
	SizeofInet6Pktinfo  = C.sizeof_struct_in6_pktinfo
	SizeofIPv6MTUInfo   = C.sizeof_struct_ip6_mtuinfo
	SizeofICMPv6Filter  = C.sizeof_struct_icmp6_filter
	SizeofUcred         = C.sizeof_struct_ucred
)

// Misc

type FdSet C.fd_set

type Sysinfo_t C.struct_sysinfo

type Utsname C.struct_utsname

type Ustat_t C.struct_ustat

const (
	_AT_FDCWD = C.AT_FDCWD
)

// Terminal handling

type Termios C.struct_termios

const (
	VINTR    = C.VINTR
	VQUIT    = C.VQUIT
	VERASE   = C.VERASE
	VKILL    = C.VKILL
	VEOF     = C.VEOF
	VTIME    = C.VTIME
	VMIN     = C.VMIN
	VSWTC    = C.VSWTC
	VSTART   = C.VSTART
	VSTOP    = C.VSTOP
	VSUSP    = C.VSUSP
	VEOL     = C.VEOL
	VREPRINT = C.VREPRINT
	VDISCARD = C.VDISCARD
	VWERASE  = C.VWERASE
	VLNEXT   = C.VLNEXT
	VEOL2    = C.VEOL2
	IGNBRK   = C.IGNBRK
	BRKINT   = C.BRKINT
	IGNPAR   = C.IGNPAR
	PARMRK   = C.PARMRK
	INPCK    = C.INPCK
	ISTRIP   = C.ISTRIP
	INLCR    = C.INLCR
	IGNCR    = C.IGNCR
	ICRNL    = C.ICRNL
	IUCLC    = C.IUCLC
	IXON     = C.IXON
	IXANY    = C.IXANY
	IXOFF    = C.IXOFF
	IMAXBEL  = C.IMAXBEL
	IUTF8    = C.IUTF8
	OPOST    = C.OPOST
	OLCUC    = C.OLCUC
	ONLCR    = C.ONLCR
	OCRNL    = C.OCRNL
	ONOCR    = C.ONOCR
	ONLRET   = C.ONLRET
	OFILL    = C.OFILL
	OFDEL    = C.OFDEL
	B0       = C.B0
	B50      = C.B50
	B75      = C.B75
	B110     = C.B110
	B134     = C.B134
	B150     = C.B150
	B200     = C.B200
	B300     = C.B300
	B600     = C.B600
	B1200    = C.B1200
	B1800    = C.B1800
	B2400    = C.B2400
	B4800    = C.B4800
	B9600    = C.B9600
	B19200   = C.B19200
	B38400   = C.B38400
	CSIZE    = C.CSIZE
	CS5      = C.CS5
	CS6      = C.CS6
	CS7      = C.CS7
	CS8      = C.CS8
	CSTOPB   = C.CSTOPB
	CREAD    = C.CREAD
	PARENB   = C.PARENB
	PARODD   = C.PARODD
	HUPCL    = C.HUPCL
	CLOCAL   = C.CLOCAL
	B57600   = C.B57600
	B115200  = C.B115200
	B230400  = C.B230400
	B460800  = C.B460800
	B500000  = C.B500000
	B576000  = C.B576000
	B921600  = C.B921600
	B1000000 = C.B1000000
	B1152000 = C.B1152000
	B1500000 = C.B1500000
	B2000000 = C.B2000000
	B2500000 = C.B2500000
	B3000000 = C.B3000000
	B3500000 = C.B3500000
	B4000000 = C.B4000000
	ISIG     = C.ISIG
	ICANON   = C.ICANON
	XCASE    = C.XCASE
	ECHO     = C.ECHO
	ECHOE    = C.ECHOE
	ECHOK    = C.ECHOK
	ECHONL   = C.ECHONL
	NOFLSH   = C.NOFLSH
	TOSTOP   = C.TOSTOP
	ECHOCTL  = C.ECHOCTL
	ECHOPRT  = C.ECHOPRT
	ECHOKE   = C.ECHOKE
	FLUSHO   = C.FLUSHO
	PENDIN   = C.PENDIN
	IEXTEN   = C.IEXTEN
	TCGETS   = C.TCGETS
	TCSETS   = C.TCSETS
)
