// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Akaros system calls.
// This file is compiled as ordinary Go code,
// but it is also input to mksyscall,
// which parses the //sys lines and generates system call stubs.
// Note that sometimes we use a lowercase //sys name and
// wrap it in our own nicer implementation.

package syscall

import (
	"runtime/parlib"
	"sync"
	"unsafe"
)

var (
	Stdin  = 0
	Stdout = 1
	Stderr = 2
)

// A Traditional Errno
type Errno uintptr
func (e Errno) Error() string {
	if 0 <= int(e) && int(e) < len(errors) {
		s := errors[e]
		if s != "" {
			return s
		}
	}
	return "Errno: " + itoa(int(e))
}
func (e Errno) Temporary() bool {
	return e == EINTR || e == EMFILE || e.Timeout()
}
func (e Errno) Timeout() bool {
	return e == EAGAIN || e == EWOULDBLOCK || e == ETIMEDOUT
}

// An AkaError is a combination of a traditional errno and a custom string.
type AkaError struct {
	errno Errno
	errstr string
}
func (e AkaError) Error() string {
	if e.errstr == "" {
		return e.errno.Error()
	}
	return e.errstr;
}
func (e AkaError) Errno() Errno {
	return e.errno
}
func (e AkaError) Errstr() string {
	return e.errstr
}
func (e AkaError) Temporary() bool {
	return e.errno.Temporary()
}
func (e AkaError) Timeout() bool {
	return e.errno == EINTR
}
func NewAkaError(errno Errno, errstr string) error { return &AkaError{errno, errstr} }

// A Signal is a number describing a process signal.
// It implements the os.Signal interface.
type Signal int
func (s Signal) Signal() {}
func (s Signal) String() string {
	if 0 <= s && int(s) < len(signals) {
		str := signals[s]
		if str != "" {
			return str
		}
	}
	return "Signal " + itoa(int(s))
}

// An Event is a number describing an Akaros event type
type Event int
func (e Event) Event() {}
func (e Event) String() string {
	return "Event " + itoa(int(e))
}

// Convert a c string to a go string
func cstring(s []byte) string {
	for i := range s {
		if s[i] == 0 {
			return string(s[0:i])
		}
	}
	return string(s)
}

type WaitStatus uint32
// Wait status is 7 bits at bottom, either 0 (exited),
// 0x7F (stopped), or a signal number that caused an exit.
// The 0x80 bit is whether there was a core dump.
// An extra number (exit code, signal causing a stop)
// is in the high bits.  At least that's the idea.
// There are various irregularities.  For example, the
// "continued" status is 0xFFFF, distinguishing itself
// from stopped via the core dump bit.
const (
	mask    = 0x7F
	core    = 0x80
	exited  = 0x00
	stopped = 0x7F
	shift   = 8
)
func (w WaitStatus) Exited() bool { return w&mask == exited }
func (w WaitStatus) Signaled() bool { return w&mask != stopped && w&mask != exited }
func (w WaitStatus) Stopped() bool { return w&0xFF == stopped }
func (w WaitStatus) Continued() bool { return w == 0xFFFF }
func (w WaitStatus) CoreDump() bool { return w.Signaled() && w&core != 0 }
func (w WaitStatus) ExitStatus() int {
	if !w.Exited() {
		return -1
	}
	return int(w>>shift) & 0xFF
}
func (w WaitStatus) Signal() Signal {
	if !w.Signaled() {
		return -1
	}
	return Signal(w & mask)
}
func (w WaitStatus) StopSignal() Signal {
	if !w.Stopped() {
		return -1
	}
	return Signal(w>>shift) & 0xFF
}
func (w WaitStatus) TrapCause() int {
	if w.StopSignal() != SIGTRAP {
		return -1
	}
	return int(w>>shift) >> 8
}

// Syscalls...
// Akaros syscalls are all made through the parlib.Syscall() interface.
// If a syscall follows the general form dictated for syscalls in mksyscall.pl,
// then we can automatically generate the syscalls using a //sys or //sysnb
// directive.  The difference between //sys and //sysnb is whether the
// generated syscall makes an underlying call to Syscall() or RawSyscall().
// Traditionally, the RawSyscall version has been used to generate NON-BLOCKING
// versions of a syscall.  In Akaros, however, ALL syscalls are non-blocking,
// so we use //sysnb for a different purpose.
// Regular syscalls check the return value (i.e. __r1 == -1) to determine if
// they should construct an AkaError and return an error. RawSyscalls check the
// value of errno (i.e. __err == 0) to determine whether an error has occured.
// In practice, most syscalls return -1 to signify an error, and may or may not
// have errno set if the call is successful (in which case, checking errno
// instead of the return value may indicate an error even if there wasn't
// one).  However, some calls (like mmap) don't return -1 on error so we need a
// different way of determining if an error occured.  Syscalls of this type
// MUST only set errno if there was actually an error, otherwise we may be
// screwed.  Seems to be fine for now. We plan to standardize how all
// error handing is doen in Akaros soon, and when that happens, having to make
// this distinction will be unnecessary.
func Syscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err error) {
	return Syscall6(trap, a1, a2, a3, 0, 0, 0)
}
func RawSyscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err error) {
	return RawSyscall6(trap, a1, a2, a3, 0, 0, 0)
}
func Syscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err error) {
	// I have syscall numbers >=300 stubbed out since they are not yet
	// implemented.  If we are trying to call one of those, print out a warning
	// and return an error.
	if trap >= 300 {
		parlib.SyscallError(trap);
		return r1, r2, EINVAL
	}

	// Otherwise, run the syscall!
	__r1, __err, __errstr := parlib.Syscall(uint32(trap), int(a1), int(a2),
	                                        int(a3), int(a4), int(a5), int(a6))

	var akaerror error = nil
	if __r1 == -1 {
		akaerror = NewAkaError(Errno(__err), string(__errstr))
	}
	return uintptr(__r1), r2, akaerror
}
func RawSyscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err error) {
	// I have syscall numbers >=300 stubbed out since they are not yet
	// implemented.  If we are trying to call one of those, print out a warning
	// and return an error.
	if trap >= 300 {
		parlib.SyscallError(trap);
		return r1, r2, EINVAL
	}

	// Otherwise, run the syscall!
	__r1, __err, __errstr := parlib.Syscall(uint32(trap), int(a1), int(a2),
	                                        int(a3), int(a4), int(a5), int(a6))

	var akaerror error = nil
	if __err != 0 {
		akaerror = NewAkaError(Errno(__err), string(__errstr))
	}
	return uintptr(__r1), r2, akaerror
}

// These syscalls can be directly generated by the mksyscall.sh script, so we
// don't need to wrap them, just list them with the //sys directive.
//sys	Close(fd int) (err error)
//sys	Read(fd int, p []byte) (n int, err error)
//sys	Write(fd int, p []byte) (n int, err error)
//sys	Block(usec int) (err error)
//sys	Fstat(fd int, stat *Stat_t) (err error)
//sys	fcntl(fd int, cmd int, arg int) (val int, err error)
//sys	AbortSyscFd(fd int) (val int, err error)
//sys	Getcwd(buf []byte, length int) (n int, err error)
//sys	Wstat(path string, pathlen int, stat_m []byte, flags int) (err error)
//sys	Fwstat(fd int, stat_m []byte, flags int) (err error)

// Locally wrapped syscalls
//sys	openat(fromfd int, path string, pathlen int, flags int, mode uint32) (fd int, err error)
func Openat(dirfd int, path string, flags int, mode uint32) (fd int, err error) {
	if len(path) == 0 {
		return -1, NewAkaError(Errno(EINVAL), "Path length 0")
	}
	return openat(dirfd, path, len(path), flags, mode)
}

func Open(path string, flags int, mode ...uint32) (fd int, err error) {
	return Openat(_AT_FDCWD, path, flags, mode[0])
}

//sys	chdir(pid int, path string, pathlen int) (err error)
func Chdir(path string) (err error) {
	if len(path) == 0 {
		return NewAkaError(Errno(EINVAL), "Path length 0")
	}
	return chdir(int(parlib.Procinfo.Pid), path, len(path))
}

//sys	fchdir(pid int, fd int) (err error)
func Fchdir(fd int) (err error) {
	return fchdir(int(parlib.Procinfo.Pid), fd)
}

//sys	llseek(fd int, offset_hi int32, offset_lo int32, result *int64, whence int) (err error)
func Seek(fd int, offset int64, whence int) (newoffset int64, err error) {
    if (fd < 0) {
        return -1, EBADF;
    }
    switch (whence) {
        case SEEK_SET:
        case SEEK_CUR:
        case SEEK_END:
            break;
        default:
			return -1, EINVAL
    }
    hi := int32(offset >> 32);
    lo := int32(offset & 0xffffffff);
    err = llseek(fd, hi, lo, &newoffset, whence);
	return newoffset, err
}

//sys	proc_destroy(pid int, exitcode int) (err error)
func Exit(exitcode int) {
	proc_destroy(int(parlib.Procinfo.Pid), exitcode)
}

func Pipe(p []int, flags int) (err error) {
	if len(p) != 2 {
		return EINVAL
	}
	dirfd, err := Open("#pipe", O_PATH | (flags & O_CLOEXEC), 0 )
	if err != nil {
		return
	}
	dfd, err := Openat(dirfd, "data", O_RDWR | flags, 0 )
	if err != nil {
		Close(dirfd);
		return
	}
	d1fd, err := Openat(dirfd, "data1", O_RDWR | flags, 0 )
	if err != nil {
		Close(dfd);
		Close(dirfd);
		return
	}
	p[0] = dfd
	p[1] = d1fd
	Close(dirfd)
	return
}

//sys	rename(oldpath string, ol int, newpath string, nl int) (err error)
func Rename(oldpath string, newpath string) (err error) {
	return rename(oldpath, len(oldpath), newpath, len(newpath))
}

//sys	unlink(path string, pathlen int) (err error)
func Unlink(path string) (err error) {
	return unlink(path, len(path))
}

//sys	rmdir(path string, pathlen int) (err error)
func Rmdir(path string) (err error) {
	return rmdir(path, len(path))
}

//sys	stat(path string, pathlen int, stat *Stat_t) (err error)
func Stat(path string, s *Stat_t) (err error) {
	return stat(path, len(path), s)
}

//sys	lstat(path string, pathlen int, stat *Stat_t) (err error)
func Lstat(path string, s *Stat_t) (err error) {
	return lstat(path, len(path), s)
}

func Pread(fd int, p []byte, offset int64) (n int, err error) {
	/* Saved offset */
	var o_offset int64

	/* Save the current offset so we can restore it later */
	o_offset, err = Seek(fd, 0, SEEK_CUR)
	if err != nil {
		return
	}

	/* Seek to wanted position.  */
	_, err = Seek(fd, offset, SEEK_SET)
	if err != nil {
		return
	}

	/* Read in the data.  */
	n, err = Read(fd, p);

	/* Seek back to the original position. If this fails, we return its error,
	 * only if the read before succedded. Otherwise we bypass this error and
	 * return the error from the read below. */
	_, __err := Seek(fd, o_offset, SEEK_SET)
	if __err != nil {
		if err == nil {
			return n, __err
		}
	}

	/* Return the result of the read */
	return
}

func Pwrite(fd int, p []byte, offset int64) (n int, err error) {
	/* Saved offset */
	var o_offset int64

	/* Save the current offset so we can restore it later */
	o_offset, err = Seek(fd, 0, SEEK_CUR)
	if err != nil {
		return
	}

	/* Seek to wanted position.  */
	_, err = Seek(fd, offset, SEEK_SET)
	if err != nil {
		return
	}

	/* Write out the data.  */
	n, err = Write(fd, p);

	/* Seek back to the original position. If this fails, we return its error,
	 * only if the read before succedded. Otherwise we bypass this error and
	 * return the error from the write below. */
	_, __err := Seek(fd, o_offset, SEEK_SET)
	if __err != nil {
		if err == nil {
			return n, __err
		}
	}

	/* Return the result of the write */
	return
}

func ReadDirent(fd int, buf []byte) (n int, err error) {
	dsize := int(unsafe.Sizeof(Dirent{}))
	n, err = Read(fd, buf[0:dsize])
	if n == 0 && err != nil {
		if err.(*AkaError).errno == ENOENT {
			err = nil
		}
	}
	return n, err
}

func ParseDirent(buf []byte, max int, names []string) (consumed int, count int, newnames []string) {
	dirent := (*Dirent)(unsafe.Pointer(&buf[0]))
	bytes := (*[256]byte)(unsafe.Pointer(&dirent.Name[0]))

	var name = string(bytes[0:clen(bytes[:])])
	if name == "." || name == ".." { // Useless names
		return len(buf), 0, names
	}
	return len(buf), 1, append(names, name)
}

func Dup(oldfd int) (fd int, err error) {
  return fcntl(oldfd, F_DUPFD, 0)
}

//sys	fd2path(fd int, buf []byte) (err error)
func Fd2path(fd int) (path string, err error) {
	var buf [512]byte

	e := fd2path(fd, buf[:])
	if e != nil {
		return "", e
	}
	return cstring(buf[:]), nil
}

// Mmap manager, for use by operating system-specific implementations.
type mmapper struct {
	sync.Mutex
	active map[*byte][]byte // active mappings; key is last byte in mapping
	mmap   func(addr, length uintptr, prot, flags, fd int, offset int64) (uintptr, error)
	munmap func(addr uintptr, length uintptr) error
}
var mapper = &mmapper{
	active: make(map[*byte][]byte),
	mmap:   mmap,
	munmap: munmap,
}
func (m *mmapper) Mmap(fd int, offset int64, length int, prot int, flags int) (data []byte, err error) {
	if length <= 0 {
		return nil, EINVAL
	}

	// Map the requested memory.
	addr, akaerror := m.mmap(0, uintptr(length), prot, flags, fd, offset)
	if akaerror != nil {
		return nil, akaerror
	}

	// Slice memory layout
	var sl = struct {
		addr uintptr
		len  int
		cap  int
	}{addr, length, length}

	// Use unsafe to turn sl into a []byte.
	b := *(*[]byte)(unsafe.Pointer(&sl))

	// Register mapping in m and return it.
	p := &b[cap(b)-1]
	m.Lock()
	defer m.Unlock()
	m.active[p] = b
	return b, nil
}
func (m *mmapper) Munmap(data []byte) (err error) {
	if len(data) == 0 || len(data) != cap(data) {
		return EINVAL
	}

	// Find the base of the mapping.
	p := &data[cap(data)-1]
	m.Lock()
	defer m.Unlock()
	b := m.active[p]
	if b == nil || &b[0] != &data[0] {
		return EINVAL
	}

	// Unmap the memory and update m.
	if akaerror := m.munmap(uintptr(unsafe.Pointer(&b[0])), uintptr(len(b))); akaerror != nil {
		return akaerror
	}
	delete(m.active, p)
	return nil
}
func Mmap(fd int, offset int64, length int, prot int, flags int) (data []byte, err error) {
	return mapper.Mmap(fd, offset, length, prot, flags)
}
//sys	munmap(addr uintptr, length uintptr) (err error)
func Munmap(b []byte) (err error) {
	return mapper.Munmap(b)
}

//sys	waitpid(pid int, wstatus *_C_int, options int) (wpid int, err error)
func Waitpid(pid int, wstatus *WaitStatus, options int) (wpid int, err error) {
	var status _C_int
	wpid, err = waitpid(pid, &status, options)
	if wstatus != nil {
		*wstatus = WaitStatus(status)
	}
	return
}
func Wait4(pid int, wstatus *WaitStatus, options int, rusage *Rusage) (wpid int, err error) {
	// We only implement this function for compatibility with unix.
	// On akaros, we simply ignore the rusage parameter for now...
	return Waitpid(pid, wstatus, options)
}

//sys	notify(pid int, ev Event, ev_msg *EventMsg) (err error)
func Kill(pid int, sig Signal) (err error) {
	localMsg := EventMsg{};
	if pid <= 0 {
		return ENOSYS
	}
	if (sig == SIGKILL) {
		return proc_destroy(pid, 0)
	}
	localMsg.Type = uint16(EV_POSIX_SIGNAL);
	localMsg.Arg1 = uint16(sig);
	return notify(pid, EV_POSIX_SIGNAL, &localMsg)
}


//sys	symlink(oldpath string, a int, newpath string, b int) (err error)
func Symlink(oldpath string, newpath string) (err error) {
	return symlink(oldpath, len(oldpath), newpath, len(newpath))
}

//sys	readlink(path string, pl int, buf []byte, bl int) (n int)
func Readlink(path string, b []byte) (int, error) {
	if err := readlink(path, len(path), b, len(b)); err < 0 {
		return -1, NewAkaError(Errno(err), "Readlink failed")
	} else {
		return err, nil
	}
}

const ImplementsGetwd = true
func Getwd() (wd string, err error) {
	var buf [PathMax]byte
	n, err := Getcwd(buf[0:], len(buf))
	if err != nil {
		return "", err
	}
	if n < 0 {
		return "", ENOTDIR
	}
	wd = cstring(buf[:])
	n = len(wd)

	// Remove the trailing slash if it's not just root '/'
	if n > 1 && wd[n-1] == '/' {
		n -= 1
	}
	return wd[0:n], err
}

func Getpid() (pid int) {
	return int(parlib.Procinfo.Pid)
}

/*****************************************************************************/
/******* Stuff below is ported, but only exists as stubs thus far ************/
/*****************************************************************************/
func Accept(fd int) (nfd int, sa Sockaddr, err error) { return }
func Accept4(fd int, flags int) (nfd int, sa Sockaddr, err error) { return }
func Bind(fd int, sa Sockaddr) (err error) { return }
func Connect(fd int, sa Sockaddr) (err error) { return }
func Socket(domain, typ, proto int) (fd int, err error) { return }
func Recvfrom(fd int, p []byte, flags int) (n int, from Sockaddr, err error) { return }
func Sendto(fd int, p []byte, flags int, to Sockaddr) (err error) { return }
func Getegid() (egid int) { return -1 }
func Geteuid() (euid int) { return -1 }
func Getgid() (gid int)   { return -1 }
func Getuid() (uid int)   { return -1 }
func Getgroups() (gids []int, err error) {
    return make([]int, 0), nil
}

/*****************************************************************************/
/****************** Stuff below hasn't been ported yet ***********************/
/*****************************************************************************/
/*****************************************************************************/
/*****************************************************************************/
/*****************************************************************************/
/*****************************************************************************/

func socketcall(call int, a0, a1, a2, a3, a4, a5 uintptr) (n int, err AkaError) { return n, err }
func rawsocketcall(call int, a0, a1, a2, a3, a4, a5 uintptr) (n int, err AkaError) { return n, err }

//sys	utimes(path string, times *[2]Timeval) (err error)

func Utimes(path string, tv []Timeval) (err error) {
	if len(tv) != 2 {
		return EINVAL
	}
	return utimes(path, (*[2]Timeval)(unsafe.Pointer(&tv[0])))
}

//sys	utimensat(dirfd int, path string, times *[2]Timespec) (err error)

func UtimesNano(path string, ts []Timespec) (err error) {
	if len(ts) != 2 {
		return EINVAL
	}
	err = utimensat(_AT_FDCWD, path, (*[2]Timespec)(unsafe.Pointer(&ts[0])))
	if err != ENOSYS {
		return err
	}
	// If the utimensat syscall isn't available (utimensat was added to Linux
	// in 2.6.22, Released, 8 July 2007) then fall back to utimes
	var tv [2]Timeval
	for i := 0; i < 2; i++ {
		tv[i].Sec = ts[i].Sec
		tv[i].Usec = ts[i].Nsec / 1000
	}
	return utimes(path, (*[2]Timeval)(unsafe.Pointer(&tv[0])))
}

//sys	futimesat(dirfd int, path *byte, times *[2]Timeval) (err error)

func Futimesat(dirfd int, path string, tv []Timeval) (err error) {
	if len(tv) != 2 {
		return EINVAL
	}
	pathp, err := BytePtrFromString(path)
	if err != nil {
		return err
	}
	return futimesat(dirfd, pathp, (*[2]Timeval)(unsafe.Pointer(&tv[0])))
}

func Futimes(fd int, tv []Timeval) (err error) {
	// Believe it or not, this is the best we can do on Linux
	// (and is what glibc does).
	return Utimes("/proc/self/fd/"+itoa(fd), tv)
}

func Setgroups(gids []int) (err error) {
	if len(gids) == 0 {
		return setgroups(0, nil)
	}

	a := make([]_Gid_t, len(gids))
	for i, v := range gids {
		a[i] = _Gid_t(v)
	}
	return setgroups(len(a), &a[0])
}

func Mkfifo(path string, mode uint32) (err error) {
	return Mknod(path, mode|S_IFIFO, 0)
}

// For testing: clients can set this flag to force
// creation of IPv6 sockets to return EAFNOSUPPORT.
var SocketDisableIPv6 bool

type Sockaddr interface {
	sockaddr() (ptr uintptr, len _Socklen, err error) // lowercase; only we can define Sockaddrs
}

type SockaddrInet4 struct {
	Port int
	Addr [4]byte
	raw  RawSockaddrInet4
}

func (sa *SockaddrInet4) sockaddr() (uintptr, _Socklen, error) {
	if sa.Port < 0 || sa.Port > 0xFFFF {
		return 0, 0, EINVAL
	}
	sa.raw.Family = AF_INET
	p := (*[2]byte)(unsafe.Pointer(&sa.raw.Port))
	p[0] = byte(sa.Port >> 8)
	p[1] = byte(sa.Port)
	for i := 0; i < len(sa.Addr); i++ {
		sa.raw.Addr[i] = sa.Addr[i]
	}
	return uintptr(unsafe.Pointer(&sa.raw)), SizeofSockaddrInet4, nil
}

type SockaddrInet6 struct {
	Port   int
	ZoneId uint32
	Addr   [16]byte
	raw    RawSockaddrInet6
}

func (sa *SockaddrInet6) sockaddr() (uintptr, _Socklen, error) {
	if sa.Port < 0 || sa.Port > 0xFFFF {
		return 0, 0, EINVAL
	}
	sa.raw.Family = AF_INET6
	p := (*[2]byte)(unsafe.Pointer(&sa.raw.Port))
	p[0] = byte(sa.Port >> 8)
	p[1] = byte(sa.Port)
	sa.raw.Scope_id = sa.ZoneId
	for i := 0; i < len(sa.Addr); i++ {
		sa.raw.Addr[i] = sa.Addr[i]
	}
	return uintptr(unsafe.Pointer(&sa.raw)), SizeofSockaddrInet6, nil
}

type SockaddrUnix struct {
	Name string
	raw  RawSockaddrUnix
}

func (sa *SockaddrUnix) sockaddr() (uintptr, _Socklen, error) {
	name := sa.Name
	n := len(name)
	if n >= len(sa.raw.Path) {
		return 0, 0, EINVAL
	}
	sa.raw.Family = AF_UNIX
	for i := 0; i < n; i++ {
		sa.raw.Path[i] = int8(name[i])
	}
	// length is family (uint16), name, NUL.
	sl := _Socklen(2)
	if n > 0 {
		sl += _Socklen(n) + 1
	}
	if sa.raw.Path[0] == '@' {
		sa.raw.Path[0] = 0
		// Don't count trailing NUL for abstract address.
		sl--
	}

	return uintptr(unsafe.Pointer(&sa.raw)), sl, nil
}

type SockaddrLinklayer struct {
	Protocol uint16
	Ifindex  int
	Hatype   uint16
	Pkttype  uint8
	Halen    uint8
	Addr     [8]byte
	raw      RawSockaddrLinklayer
}

func (sa *SockaddrLinklayer) sockaddr() (uintptr, _Socklen, error) {
	if sa.Ifindex < 0 || sa.Ifindex > 0x7fffffff {
		return 0, 0, EINVAL
	}
	sa.raw.Family = AF_PACKET
	sa.raw.Protocol = sa.Protocol
	sa.raw.Ifindex = int32(sa.Ifindex)
	sa.raw.Hatype = sa.Hatype
	sa.raw.Pkttype = sa.Pkttype
	sa.raw.Halen = sa.Halen
	for i := 0; i < len(sa.Addr); i++ {
		sa.raw.Addr[i] = sa.Addr[i]
	}
	return uintptr(unsafe.Pointer(&sa.raw)), SizeofSockaddrLinklayer, nil
}

type SockaddrNetlink struct {
	Family uint16
	Pad    uint16
	Pid    uint32
	Groups uint32
	raw    RawSockaddrNetlink
}

func (sa *SockaddrNetlink) sockaddr() (uintptr, _Socklen, error) {
	sa.raw.Family = AF_NETLINK
	sa.raw.Pad = sa.Pad
	sa.raw.Pid = sa.Pid
	sa.raw.Groups = sa.Groups
	return uintptr(unsafe.Pointer(&sa.raw)), SizeofSockaddrNetlink, nil
}

func anyToSockaddr(rsa *RawSockaddrAny) (Sockaddr, error) {
	switch rsa.Addr.Family {
	case AF_NETLINK:
		pp := (*RawSockaddrNetlink)(unsafe.Pointer(rsa))
		sa := new(SockaddrNetlink)
		sa.Family = pp.Family
		sa.Pad = pp.Pad
		sa.Pid = pp.Pid
		sa.Groups = pp.Groups
		return sa, nil

	case AF_PACKET:
		pp := (*RawSockaddrLinklayer)(unsafe.Pointer(rsa))
		sa := new(SockaddrLinklayer)
		sa.Protocol = pp.Protocol
		sa.Ifindex = int(pp.Ifindex)
		sa.Hatype = pp.Hatype
		sa.Pkttype = pp.Pkttype
		sa.Halen = pp.Halen
		for i := 0; i < len(sa.Addr); i++ {
			sa.Addr[i] = pp.Addr[i]
		}
		return sa, nil

	case AF_UNIX:
		pp := (*RawSockaddrUnix)(unsafe.Pointer(rsa))
		sa := new(SockaddrUnix)
		if pp.Path[0] == 0 {
			// "Abstract" Unix domain socket.
			// Rewrite leading NUL as @ for textual display.
			// (This is the standard convention.)
			// Not friendly to overwrite in place,
			// but the callers below don't care.
			pp.Path[0] = '@'
		}

		// Assume path ends at NUL.
		// This is not technically the Linux semantics for
		// abstract Unix domain sockets--they are supposed
		// to be uninterpreted fixed-size binary blobs--but
		// everyone uses this convention.
		n := 0
		for n < len(pp.Path) && pp.Path[n] != 0 {
			n++
		}
		bytes := (*[10000]byte)(unsafe.Pointer(&pp.Path[0]))[0:n]
		sa.Name = string(bytes)
		return sa, nil

	case AF_INET:
		pp := (*RawSockaddrInet4)(unsafe.Pointer(rsa))
		sa := new(SockaddrInet4)
		p := (*[2]byte)(unsafe.Pointer(&pp.Port))
		sa.Port = int(p[0])<<8 + int(p[1])
		for i := 0; i < len(sa.Addr); i++ {
			sa.Addr[i] = pp.Addr[i]
		}
		return sa, nil

	case AF_INET6:
		pp := (*RawSockaddrInet6)(unsafe.Pointer(rsa))
		sa := new(SockaddrInet6)
		p := (*[2]byte)(unsafe.Pointer(&pp.Port))
		sa.Port = int(p[0])<<8 + int(p[1])
		sa.ZoneId = pp.Scope_id
		for i := 0; i < len(sa.Addr); i++ {
			sa.Addr[i] = pp.Addr[i]
		}
		return sa, nil
	}
	return nil, EAFNOSUPPORT
}

func Getsockname(fd int) (sa Sockaddr, err error) {
	var rsa RawSockaddrAny
	var len _Socklen = SizeofSockaddrAny
	if err = getsockname(fd, &rsa, &len); err != nil {
		return
	}
	return anyToSockaddr(&rsa)
}

func Getpeername(fd int) (sa Sockaddr, err error) {
	var rsa RawSockaddrAny
	var len _Socklen = SizeofSockaddrAny
	if err = getpeername(fd, &rsa, &len); err != nil {
		return
	}
	return anyToSockaddr(&rsa)
}

func Socketpair(domain, typ, proto int) (fd [2]int, err error) {
	var fdx [2]int32
	err = socketpair(domain, typ, proto, &fdx)
	if err == nil {
		fd[0] = int(fdx[0])
		fd[1] = int(fdx[1])
	}
	return
}

func GetsockoptInt(fd, level, opt int) (value int, err error) {
	var n int32
	vallen := _Socklen(4)
	err = getsockopt(fd, level, opt, uintptr(unsafe.Pointer(&n)), &vallen)
	return int(n), err
}

func GetsockoptInet4Addr(fd, level, opt int) (value [4]byte, err error) {
	vallen := _Socklen(4)
	err = getsockopt(fd, level, opt, uintptr(unsafe.Pointer(&value[0])), &vallen)
	return value, err
}

func GetsockoptIPMreq(fd, level, opt int) (*IPMreq, error) {
	var value IPMreq
	vallen := _Socklen(SizeofIPMreq)
	err := getsockopt(fd, level, opt, uintptr(unsafe.Pointer(&value)), &vallen)
	return &value, err
}

func GetsockoptIPMreqn(fd, level, opt int) (*IPMreqn, error) {
	var value IPMreqn
	vallen := _Socklen(SizeofIPMreqn)
	err := getsockopt(fd, level, opt, uintptr(unsafe.Pointer(&value)), &vallen)
	return &value, err
}

func GetsockoptIPv6Mreq(fd, level, opt int) (*IPv6Mreq, error) {
	var value IPv6Mreq
	vallen := _Socklen(SizeofIPv6Mreq)
	err := getsockopt(fd, level, opt, uintptr(unsafe.Pointer(&value)), &vallen)
	return &value, err
}

func GetsockoptIPv6MTUInfo(fd, level, opt int) (*IPv6MTUInfo, error) {
	var value IPv6MTUInfo
	vallen := _Socklen(SizeofIPv6MTUInfo)
	err := getsockopt(fd, level, opt, uintptr(unsafe.Pointer(&value)), &vallen)
	return &value, err
}

func GetsockoptICMPv6Filter(fd, level, opt int) (*ICMPv6Filter, error) {
	var value ICMPv6Filter
	vallen := _Socklen(SizeofICMPv6Filter)
	err := getsockopt(fd, level, opt, uintptr(unsafe.Pointer(&value)), &vallen)
	return &value, err
}

func GetsockoptUcred(fd, level, opt int) (*Ucred, error) {
	var value Ucred
	vallen := _Socklen(SizeofUcred)
	err := getsockopt(fd, level, opt, uintptr(unsafe.Pointer(&value)), &vallen)
	return &value, err
}

func SetsockoptInt(fd, level, opt int, value int) (err error) {
	var n = int32(value)
	return setsockopt(fd, level, opt, uintptr(unsafe.Pointer(&n)), 4)
}

func SetsockoptInet4Addr(fd, level, opt int, value [4]byte) (err error) {
	return setsockopt(fd, level, opt, uintptr(unsafe.Pointer(&value[0])), 4)
}

func SetsockoptTimeval(fd, level, opt int, tv *Timeval) (err error) {
	return setsockopt(fd, level, opt, uintptr(unsafe.Pointer(tv)), unsafe.Sizeof(*tv))
}

func SetsockoptLinger(fd, level, opt int, l *Linger) (err error) {
	return setsockopt(fd, level, opt, uintptr(unsafe.Pointer(l)), unsafe.Sizeof(*l))
}

func SetsockoptIPMreq(fd, level, opt int, mreq *IPMreq) (err error) {
	return setsockopt(fd, level, opt, uintptr(unsafe.Pointer(mreq)), unsafe.Sizeof(*mreq))
}

func SetsockoptIPMreqn(fd, level, opt int, mreq *IPMreqn) (err error) {
	return setsockopt(fd, level, opt, uintptr(unsafe.Pointer(mreq)), unsafe.Sizeof(*mreq))
}

func SetsockoptIPv6Mreq(fd, level, opt int, mreq *IPv6Mreq) (err error) {
	return setsockopt(fd, level, opt, uintptr(unsafe.Pointer(mreq)), unsafe.Sizeof(*mreq))
}

func SetsockoptICMPv6Filter(fd, level, opt int, filter *ICMPv6Filter) error {
	return setsockopt(fd, level, opt, uintptr(unsafe.Pointer(filter)), SizeofICMPv6Filter)
}
func SetsockoptString(fd, level, opt int, s string) (err error) {
	return setsockopt(fd, level, opt, uintptr(unsafe.Pointer(&[]byte(s)[0])), uintptr(len(s)))
}

func Recvmsg(fd int, p, oob []byte, flags int) (n, oobn int, recvflags int, from Sockaddr, err error) {
	var msg Msghdr
	var rsa RawSockaddrAny
	msg.Name = (*byte)(unsafe.Pointer(&rsa))
	msg.Namelen = uint32(SizeofSockaddrAny)
	var iov Iovec
	if len(p) > 0 {
		iov.Base = (*byte)(unsafe.Pointer(&p[0]))
		iov.SetLen(len(p))
	}
	var dummy byte
	if len(oob) > 0 {
		// receive at least one normal byte
		if len(p) == 0 {
			iov.Base = &dummy
			iov.SetLen(1)
		}
		msg.Control = (*byte)(unsafe.Pointer(&oob[0]))
		msg.SetControllen(len(oob))
	}
	msg.Iov = &iov
	msg.Iovlen = 1
	if n, err = recvmsg(fd, &msg, flags); err != nil {
		return
	}
	oobn = int(msg.Controllen)
	recvflags = int(msg.Flags)
	// source address is only specified if the socket is unconnected
	if rsa.Addr.Family != AF_UNSPEC {
		from, err = anyToSockaddr(&rsa)
	}
	return
}

func Sendmsg(fd int, p, oob []byte, to Sockaddr, flags int) (err error) {
	var ptr uintptr
	var salen _Socklen
	if to != nil {
		var err error
		ptr, salen, err = to.sockaddr()
		if err != nil {
			return err
		}
	}
	var msg Msghdr
	msg.Name = (*byte)(unsafe.Pointer(ptr))
	msg.Namelen = uint32(salen)
	var iov Iovec
	if len(p) > 0 {
		iov.Base = (*byte)(unsafe.Pointer(&p[0]))
		iov.SetLen(len(p))
	}
	var dummy byte
	if len(oob) > 0 {
		// send at least one normal byte
		if len(p) == 0 {
			iov.Base = &dummy
			iov.SetLen(1)
		}
		msg.Control = (*byte)(unsafe.Pointer(&oob[0]))
		msg.SetControllen(len(oob))
	}
	msg.Iov = &iov
	msg.Iovlen = 1
	if err = sendmsg(fd, &msg, flags); err != nil {
		return
	}
	return
}

// BindToDevice binds the socket associated with fd to device.
func BindToDevice(fd int, device string) (err error) {
	return SetsockoptString(fd, SOL_SOCKET, SO_BINDTODEVICE, device)
}

//sys	ptrace(request int, pid int, addr uintptr, data uintptr) (err error)

func ptracePeek(req int, pid int, addr uintptr, out []byte) (count int, err error) {
	// The peek requests are machine-size oriented, so we wrap it
	// to retrieve arbitrary-length data.

	// The ptrace syscall differs from glibc's ptrace.
	// Peeks returns the word in *data, not as the return value.

	var buf [sizeofPtr]byte

	// Leading edge.  PEEKTEXT/PEEKDATA don't require aligned
	// access (PEEKUSER warns that it might), but if we don't
	// align our reads, we might straddle an unmapped page
	// boundary and not get the bytes leading up to the page
	// boundary.
	n := 0
	if addr%sizeofPtr != 0 {
		err = ptrace(req, pid, addr-addr%sizeofPtr, uintptr(unsafe.Pointer(&buf[0])))
		if err != nil {
			return 0, err
		}
		n += copy(out, buf[addr%sizeofPtr:])
		out = out[n:]
	}

	// Remainder.
	for len(out) > 0 {
		// We use an internal buffer to guarantee alignment.
		// It's not documented if this is necessary, but we're paranoid.
		err = ptrace(req, pid, addr+uintptr(n), uintptr(unsafe.Pointer(&buf[0])))
		if err != nil {
			return n, err
		}
		copied := copy(out, buf[0:])
		n += copied
		out = out[copied:]
	}

	return n, nil
}

func PtracePeekText(pid int, addr uintptr, out []byte) (count int, err error) {
	return ptracePeek(PTRACE_PEEKTEXT, pid, addr, out)
}

func PtracePeekData(pid int, addr uintptr, out []byte) (count int, err error) {
	return ptracePeek(PTRACE_PEEKDATA, pid, addr, out)
}

func ptracePoke(pokeReq int, peekReq int, pid int, addr uintptr, data []byte) (count int, err error) {
	// As for ptracePeek, we need to align our accesses to deal
	// with the possibility of straddling an invalid page.

	// Leading edge.
	n := 0
	if addr%sizeofPtr != 0 {
		var buf [sizeofPtr]byte
		err = ptrace(peekReq, pid, addr-addr%sizeofPtr, uintptr(unsafe.Pointer(&buf[0])))
		if err != nil {
			return 0, err
		}
		n += copy(buf[addr%sizeofPtr:], data)
		word := *((*uintptr)(unsafe.Pointer(&buf[0])))
		err = ptrace(pokeReq, pid, addr-addr%sizeofPtr, word)
		if err != nil {
			return 0, err
		}
		data = data[n:]
	}

	// Interior.
	for len(data) > sizeofPtr {
		word := *((*uintptr)(unsafe.Pointer(&data[0])))
		err = ptrace(pokeReq, pid, addr+uintptr(n), word)
		if err != nil {
			return n, err
		}
		n += sizeofPtr
		data = data[sizeofPtr:]
	}

	// Trailing edge.
	if len(data) > 0 {
		var buf [sizeofPtr]byte
		err = ptrace(peekReq, pid, addr+uintptr(n), uintptr(unsafe.Pointer(&buf[0])))
		if err != nil {
			return n, err
		}
		copy(buf[0:], data)
		word := *((*uintptr)(unsafe.Pointer(&buf[0])))
		err = ptrace(pokeReq, pid, addr+uintptr(n), word)
		if err != nil {
			return n, err
		}
		n += len(data)
	}

	return n, nil
}

func PtracePokeText(pid int, addr uintptr, data []byte) (count int, err error) {
	return ptracePoke(PTRACE_POKETEXT, PTRACE_PEEKTEXT, pid, addr, data)
}

func PtracePokeData(pid int, addr uintptr, data []byte) (count int, err error) {
	return ptracePoke(PTRACE_POKEDATA, PTRACE_PEEKDATA, pid, addr, data)
}

func PtraceGetRegs(pid int, regsout *PtraceRegs) (err error) {
	return ptrace(PTRACE_GETREGS, pid, 0, uintptr(unsafe.Pointer(regsout)))
}

func PtraceSetRegs(pid int, regs *PtraceRegs) (err error) {
	return ptrace(PTRACE_SETREGS, pid, 0, uintptr(unsafe.Pointer(regs)))
}

func PtraceSetOptions(pid int, options int) (err error) {
	return ptrace(PTRACE_SETOPTIONS, pid, 0, uintptr(options))
}

func PtraceGetEventMsg(pid int) (msg uint, err error) {
	var data _C_long
	err = ptrace(PTRACE_GETEVENTMSG, pid, 0, uintptr(unsafe.Pointer(&data)))
	msg = uint(data)
	return
}

func PtraceCont(pid int, signal int) (err error) {
	return ptrace(PTRACE_CONT, pid, 0, uintptr(signal))
}

func PtraceSyscall(pid int, signal int) (err error) {
	return ptrace(PTRACE_SYSCALL, pid, 0, uintptr(signal))
}

func PtraceSingleStep(pid int) (err error) { return ptrace(PTRACE_SINGLESTEP, pid, 0, 0) }

func PtraceAttach(pid int) (err error) { return ptrace(PTRACE_ATTACH, pid, 0, 0) }

func PtraceDetach(pid int) (err error) { return ptrace(PTRACE_DETACH, pid, 0, 0) }

//sys	reboot(magic1 uint, magic2 uint, cmd int, arg string) (err error)

func Reboot(cmd int) (err error) {
	return reboot(LINUX_REBOOT_MAGIC1, LINUX_REBOOT_MAGIC2, cmd, "")
}

func clen(n []byte) int {
	for i := 0; i < len(n); i++ {
		if n[i] == 0 {
			return i
		}
	}
	return len(n)
}

//sys	mount(source string, target string, fstype string, flags uintptr, data *byte) (err error)

func Mount(source string, target string, fstype string, flags uintptr, data string) (err error) {
	// Certain file systems get rather angry and EINVAL if you give
	// them an empty string of data, rather than NULL.
	if data == "" {
		return mount(source, target, fstype, flags, nil)
	}
	datap, err := BytePtrFromString(data)
	if err != nil {
		return err
	}
	return mount(source, target, fstype, flags, datap)
}

// Sendto
// Recvfrom
// Socketpair

/*
 * Direct access
 */
//sys	Access(path string, mode uint32) (err error)
//sys	Acct(path string) (err error)
//sys	Adjtimex(buf *Timex) (state int, err error)
//sys	Chroot(path string) (err error)
//sys	Creat(path string, mode uint32) (fd int, err error)
//sysnb	Dup2(oldfd int, newfd int) (err error)
//sysnb	EpollCreate(size int) (fd int, err error)
//sysnb	EpollCreate1(flag int) (fd int, err error)
//sysnb	EpollCtl(epfd int, op int, fd int, event *EpollEvent) (err error)
//sys	EpollWait(epfd int, events []EpollEvent, msec int) (n int, err error)
//sys	Faccessat(dirfd int, path string, mode uint32, flags int) (err error)
//sys	Fallocate(fd int, mode uint32, off int64, len int64) (err error)
//sys	Fchmodat(dirfd int, path string, mode uint32, flags int) (err error)
//sys	Fchownat(dirfd int, path string, uid int, gid int, flags int) (err error)
//sys	Fdatasync(fd int) (err error)
//sys	Flock(fd int, how int) (err error)
//sys	Fsync(fd int) (err error)
//sysnb	Getpgid(pid int) (pgid int, err error)
//sysnb	Getpgrp() (pid int)
//sysnb	Getppid() (ppid int)
//sys	Getpriority(which int, who int) (prio int, err error)
//sysnb	Getrusage(who int, rusage *Rusage) (err error)
//sysnb	Gettid() (tid int)
//sys	Getxattr(path string, attr string, dest []byte) (sz int, err error)
//sys	InotifyAddWatch(fd int, pathname string, mask uint32) (watchdesc int, err error)
//sysnb	InotifyInit() (fd int, err error)
//sysnb	InotifyInit1(flags int) (fd int, err error)
//sysnb	InotifyRmWatch(fd int, watchdesc uint32) (success int, err error)
//sys	Klogctl(typ int, buf []byte) (n int, err error) = SYS_SYSLOG
//sys	Link(oldpath string, newpath string) (err error)
//sys	Listxattr(path string, dest []byte) (sz int, err error)
//sys	Mkdir(path string, mode uint32) (err error)
//sys	Mkdirat(dirfd int, path string, mode uint32) (err error)
//sys	Mknod(path string, mode uint32, dev int) (err error)
//sys	Mknodat(dirfd int, path string, mode uint32, dev int) (err error)
//sys	Nanosleep(time *Timespec, leftover *Timespec) (err error)
//sys	Pause() (err error)
//sys	PivotRoot(newroot string, putold string) (err error) = SYS_PIVOT_ROOT
//sysnb prlimit(pid int, resource int, old *Rlimit, newlimit *Rlimit) (err error) = SYS_PRLIMIT64
//sys	Removexattr(path string, attr string) (err error)
//sys	Renameat(olddirfd int, oldpath string, newdirfd int, newpath string) (err error)
//sys	Setdomainname(p []byte) (err error)
//sys	Sethostname(p []byte) (err error)
//sysnb	Setpgid(pid int, pgid int) (err error)
//sysnb	Setsid() (pid int, err error)
//sysnb	Settimeofday(tv *Timeval) (err error)
//sysnb	Setuid(uid int) (err error)
//sys	Setpriority(which int, who int, prio int) (err error)
//sys	Setxattr(path string, attr string, data []byte, flags int) (err error)
//sys	Sync()
//sysnb	Sysinfo(info *Sysinfo_t) (err error)
//sys	Tee(rfd int, wfd int, len int, flags int) (n int64, err error)
//sysnb	Tgkill(tgid int, tid int, sig Signal) (err error)
//sysnb	Times(tms *Tms) (ticks uintptr, err error)
//sysnb	Umask(mask int) (oldmask int)
//sysnb	Uname(buf *Utsname) (err error)
//sys	Unlinkat(dirfd int, path string) (err error)
//sys	Unmount(target string, flags int) (err error) = SYS_UMOUNT2
//sys	Unshare(flags int) (err error)
//sys	Ustat(dev int, ubuf *Ustat_t) (err error)
//sys	Utime(path string, buf *Utimbuf) (err error)
//sys	exitThread(code int) (err error) = SYS_EXIT
//sys	readlen(fd int, p *byte, np int) (n int, err error) = SYS_READ
//sys	writelen(fd int, p *byte, np int) (n int, err error) = SYS_WRITE
//sys	Madvise(b []byte, advice int) (err error)
//sys	Mprotect(b []byte, prot int) (err error)
//sys	Mlock(b []byte) (err error)
//sys	Munlock(b []byte) (err error)
//sys	Mlockall(flags int) (err error)
//sys	Munlockall() (err error)

/*
 * Unimplemented
 */
// AddKey
// AfsSyscall
// Alarm
// ArchPrctl
// Brk
// Capget
// Capset
// ClockGetres
// ClockGettime
// ClockNanosleep
// ClockSettime
// Clone
// CreateModule
// DeleteModule
// EpollCtlOld
// EpollPwait
// EpollWaitOld
// Eventfd
// Execve
// Fadvise64
// Fgetxattr
// Flistxattr
// Fork
// Fremovexattr
// Fsetxattr
// Futex
// GetKernelSyms
// GetMempolicy
// GetRobustList
// GetThreadArea
// Getitimer
// Getpmsg
// IoCancel
// IoDestroy
// IoGetevents
// IoSetup
// IoSubmit
// Ioctl
// IoprioGet
// IoprioSet
// KexecLoad
// Keyctl
// Lgetxattr
// Llistxattr
// LookupDcookie
// Lremovexattr
// Lsetxattr
// Mbind
// MigratePages
// Mincore
// ModifyLdt
// Mount
// MovePages
// Mprotect
// MqGetsetattr
// MqNotify
// MqOpen
// MqTimedreceive
// MqTimedsend
// MqUnlink
// Mremap
// Msgctl
// Msgget
// Msgrcv
// Msgsnd
// Msync
// Newfstatat
// Nfsservctl
// Personality
// Poll
// Ppoll
// Prctl
// Pselect6
// Ptrace
// Putpmsg
// QueryModule
// Quotactl
// Readahead
// Readv
// RemapFilePages
// RequestKey
// RestartSyscall
// RtSigaction
// RtSigpending
// RtSigprocmask
// RtSigqueueinfo
// RtSigreturn
// RtSigsuspend
// RtSigtimedwait
// SchedGetPriorityMax
// SchedGetPriorityMin
// SchedGetaffinity
// SchedGetparam
// SchedGetscheduler
// SchedRrGetInterval
// SchedSetaffinity
// SchedSetparam
// SchedYield
// Security
// Semctl
// Semget
// Semop
// Semtimedop
// SetMempolicy
// SetRobustList
// SetThreadArea
// SetTidAddress
// Shmat
// Shmctl
// Shmdt
// Shmget
// Sigaltstack
// Signalfd
// Swapoff
// Swapon
// Sysfs
// TimerCreate
// TimerDelete
// TimerGetoverrun
// TimerGettime
// TimerSettime
// Timerfd
// Tkill (obsolete)
// Tuxcall
// Umount2
// Uselib
// Utimensat
// Vfork
// Vhangup
// Vmsplice
// Vserver
// Waitid
// _Sysctl
