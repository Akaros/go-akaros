// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import (
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
)

// File represents an open file descriptor.
type File struct {
	*file
}

// file is the real representation of *File.
// The extra level of indirection ensures that no clients of os
// can overwrite this data, which could cause the finalizer
// to close the wrong file descriptor.
type file struct {
	fd      int
	name    string
	dirinfo *dirInfo // nil unless directory being read
	nepipe  int32    // number of consecutive EPIPE in Write
	iocount int32    // Count of outstanding I/O on this file
}

// Fd returns the integer Unix file descriptor referencing the open file.
func (f *File) Fd() uintptr {
	if f == nil {
		return ^(uintptr(0))
	}
	return uintptr(f.fd)
}

// NewFile returns a new File with the given file descriptor and name.
func NewFile(fd uintptr, name string) *File {
	fdi := int(fd)
	if fdi < 0 {
		return nil
	}
	f := &File{&file{fd: fdi, name: name}}
	runtime.SetFinalizer(f.file, (*file).close)
	return f
}

// Auxiliary information if the File describes a directory
type dirInfo struct {
	buf  []byte // buffer for directory I/O
	nbuf int    // length of buf; return value from Getdirentries
	bufp int    // location of next record in buf.
}

func sigpipe() // implemented in package runtime
func epipecheck(file *File, e error) {
	if e == syscall.EPIPE {
		if atomic.AddInt32(&file.nepipe, 1) >= 10 {
			sigpipe()
		}
	} else {
		atomic.StoreInt32(&file.nepipe, 0)
	}
}

// DevNull is the name of the operating system's ``null device.''
// On Unix-like systems, it is "/dev/null"; on Windows, "NUL".
const DevNull = "/dev/null"

// syscallMode returns the syscall-specific mode bits from Go's portable mode bits.
func syscallMode(i FileMode) (o uint32) {
	o |= uint32(i.Perm())
	if i&ModeSetuid != 0 {
		o |= syscall.S_ISUID
	}
	if i&ModeSetgid != 0 {
		o |= syscall.S_ISGID
	}
	if i&ModeSticky != 0 {
		o |= syscall.S_ISVTX
	}
	// No mapping for Go's ModeTemporary (plan9 only).
	return
}

// OpenFile is the generalized open call; most users will use Open
// or Create instead.  It opens the named file with specified flag
// (O_RDONLY etc.) and perm, (0666 etc.) if applicable.  If successful,
// methods on the returned File can be used for I/O.
// If there is an error, it will be of type *PathError.
func OpenFile(name string, flag int, perm FileMode) (file *File, err error) {
	r, e := syscall.Open(name, flag|syscall.O_CLOEXEC, syscallMode(perm))
	if e != nil {
		return nil, &PathError{"open", name, e}
	}

	// There's a race here with fork/exec, which we are
	// content to live with.  See ../syscall/exec_unix.go.
	if !supportsCloseOnExec {
		syscall.CloseOnExec(r)
	}

	return NewFile(uintptr(r), name), nil
}

// Close closes the File, rendering it unusable for I/O.
// It returns an error, if any.
func (f *File) Close() error {
	if f == nil {
		return ErrInvalid
	}
	return f.file.close()
}

func (file *file) close() error {
	if file == nil || file.fd < 0 {
		return syscall.EINVAL
	}
	var err error
	if e := syscall.Close(file.fd); e != nil {
		err = &PathError{"close", file.name, e}
	}
	file.fd = -1 // so it can't be closed again

	// no need for a finalizer anymore
	runtime.SetFinalizer(file, nil)
	return err
}

// Stat returns the FileInfo structure describing file.
// If there is an error, it will be of type *PathError.
func (f *File) Stat() (fi FileInfo, err error) {
	if f == nil {
		return nil, ErrInvalid
	}
	var stat syscall.Stat_t
	atomic.AddInt32(&f.file.iocount, 1)
	err = syscall.Fstat(f.fd, &stat)
	atomic.AddInt32(&f.file.iocount, -1)
	if err != nil {
		return nil, &PathError{"stat", f.name, err}
	}
	return fileInfoFromStat(&stat, f.name), nil
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func Stat(name string) (fi FileInfo, err error) {
	var stat syscall.Stat_t
	err = syscall.Stat(name, &stat)
	if err != nil {
		return nil, &PathError{"stat", name, err}
	}
	return fileInfoFromStat(&stat, name), nil
}

// Lstat returns a FileInfo describing the named file.
// If the file is a symbolic link, the returned FileInfo
// describes the symbolic link.  Lstat makes no attempt to follow the link.
// If there is an error, it will be of type *PathError.
func Lstat(name string) (fi FileInfo, err error) {
	var stat syscall.Stat_t
	err = syscall.Lstat(name, &stat)
	if err != nil {
		return nil, &PathError{"lstat", name, err}
	}
	return fileInfoFromStat(&stat, name), nil
}

func (f *File) readdir(n int) (fi []FileInfo, err error) {
	dirname := f.name
	if dirname == "" {
		dirname = "."
	}
	names, err := f.Readdirnames(n)
	fi = make([]FileInfo, 0, len(names))
	for _, filename := range names {
		fip, lerr := lstat(dirname + "/" + filename)
		if IsNotExist(lerr) {
			// File disappeared between readdir + stat.
			// Just treat it as if it didn't exist.
			continue
		}
		if lerr != nil {
			return fi, lerr
		}
		fi = append(fi, fip)
	}
	return fi, err
}

// Darwin and FreeBSD can't read or write 2GB+ at a time,
// even on 64-bit systems. See golang.org/issue/7812.
// Use 1GB instead of, say, 2GB-1, to keep subsequent
// reads aligned.
const (
	needsMaxRW = runtime.GOOS == "darwin" || runtime.GOOS == "freebsd"
	maxRW      = 1 << 30
)

// read reads up to len(b) bytes from the File.
// It returns the number of bytes read and an error, if any.
func (f *File) read(b []byte) (n int, err error) {
	if needsMaxRW && len(b) > maxRW {
		b = b[:maxRW]
	}
	atomic.AddInt32(&f.file.iocount, 1)
	n, err = syscall.Read(f.fd, b)
	atomic.AddInt32(&f.file.iocount, -1)
	return
}

// pread reads len(b) bytes from the File starting at byte offset off.
// It returns the number of bytes read and the error, if any.
// EOF is signaled by a zero count with err set to nil.
func (f *File) pread(b []byte, off int64) (n int, err error) {
	if needsMaxRW && len(b) > maxRW {
		b = b[:maxRW]
	}
	atomic.AddInt32(&f.file.iocount, 1)
	n, err = syscall.Pread(f.fd, b, off)
	atomic.AddInt32(&f.file.iocount, -1)
	return
}

// write writes len(b) bytes to the File.
// It returns the number of bytes written and an error, if any.
func (f *File) write(b []byte) (n int, err error) {
	for {
		bcap := b
		if needsMaxRW && len(bcap) > maxRW {
			bcap = bcap[:maxRW]
		}
		atomic.AddInt32(&f.file.iocount, 1)
		m, err := syscall.Write(f.fd, bcap)
		atomic.AddInt32(&f.file.iocount, -1)
		n += m

		// If the syscall wrote some data but not all (short write)
		// or it returned EINTR, then assume it stopped early for
		// reasons that are uninteresting to the caller, and try again.
		if 0 < m && m < len(bcap) || err == syscall.EINTR {
			b = b[m:]
			continue
		}

		if needsMaxRW && len(bcap) != len(b) && err == nil {
			b = b[m:]
			continue
		}

		return n, err
	}
}

// pwrite writes len(b) bytes to the File starting at byte offset off.
// It returns the number of bytes written and an error, if any.
func (f *File) pwrite(b []byte, off int64) (n int, err error) {
	if needsMaxRW && len(b) > maxRW {
		b = b[:maxRW]
	}
	atomic.AddInt32(&f.file.iocount, 1)
	n, err = syscall.Pwrite(f.fd, b, off)
	atomic.AddInt32(&f.file.iocount, -1)
	return
}

// seek sets the offset for the next Read or Write on file to offset, interpreted
// according to whence: 0 means relative to the origin of the file, 1 means
// relative to the current offset, and 2 means relative to the end.
// It returns the new offset and an error, if any.
func (f *File) seek(offset int64, whence int) (ret int64, err error) {
	atomic.AddInt32(&f.file.iocount, 1)
	ret, err = syscall.Seek(f.fd, offset, whence)
	atomic.AddInt32(&f.file.iocount, -1)
	return
}

// Truncate changes the size of the named file.
// If the file is a symbolic link, it changes the size of the link's target.
// If there is an error, it will be of type *PathError.
func Truncate(name string, size int64) error {
	var d syscall.Dir

	d.Null()
	d.Length = size

	var buf [syscall.STATFIXLEN]byte
	n, err := d.Marshal(buf[:])
	if err != nil {
		return &PathError{"truncate", name, err}
	}
	if err = syscall.Wstat(name, buf[:n], syscall.WSTAT_LENGTH); err != nil {
		return &PathError{"truncate", name, err}
	}
	return nil
}

// Truncate changes the size of the named file.
// If the file is a symbolic link, it changes the size of the link's target.
// If there is an error, it will be of type *PathError.
func (f *File) Truncate(size int64) error {
	if f == nil {
		return ErrInvalid
	}

	var d syscall.Dir
	d.Null()
	d.Length = size

	var buf [syscall.STATFIXLEN]byte
	n, err := d.Marshal(buf[:])
	if err != nil {
		return &PathError{"truncate", f.name, err}
	}
	if err = syscall.Fwstat(f.fd, buf[:n], syscall.WSTAT_LENGTH); err != nil {
		return &PathError{"truncate", f.name, err}
	}
	return nil
}

// Remove removes the named file or directory.
// If there is an error, it will be of type *PathError.
func Remove(name string) error {
	// System call interface forces us to know
	// whether name is a file or directory.
	// Try both: it is cheaper on average than
	// doing a Stat plus the right one.
	e := syscall.Unlink(name)
	if e == nil {
		return nil
	}
	e1 := syscall.Rmdir(name)
	if e1 == nil {
		return nil
	}

	// Both failed: figure out which error to return.
	// OS X and Linux differ on whether unlink(dir)
	// returns EISDIR, so can't use that.  However,
	// both agree that rmdir(file) returns ENOTDIR,
	// so we can use that to decide which error is real.
	// Rmdir might also return ENOTDIR if given a bad
	// file path, like /etc/passwd/foo, but in that case,
	// both errors will be ENOTDIR, so it's okay to
	// use the error from unlink.
	if e1 != syscall.ENOTDIR {
		e = e1
	}
	return &PathError{"remove", name, e}
}

// Link creates newname as a hard link to the oldname file.
// If there is an error, it will be of type *LinkError.
func Link(oldname, newname string) error {
	e := syscall.Link(oldname, newname)
	if e != nil {
		return &LinkError{"link", oldname, newname, e}
	}
	return nil
}

// Symlink creates newname as a symbolic link to oldname.
// If there is an error, it will be of type *LinkError.
func Symlink(oldname, newname string) error {
	e := syscall.Symlink(oldname, newname)
	if e != nil {
		return &LinkError{"symlink", oldname, newname, e}
	}
	return nil
}

// Readlink returns the destination of the named symbolic link.
// If there is an error, it will be of type *PathError.
func Readlink(name string) (string, error) {
	for len := 128; ; len *= 2 {
		b := make([]byte, len)
		n, e := syscall.Readlink(name, b)
		if e != nil {
			return "", &PathError{"readlink", name, e}
		}
		if n < len {
			return string(b[0:n]), nil
		}
	}
}

// HasPrefix from the strings package.
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

// Variant of LastIndex from the strings package.
func lastIndex(s string, sep byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == sep {
			return i
		}
	}
	return -1
}

func rename(oldname, newname string) error {
	e := syscall.Rename(oldname, newname)
	if e != nil {
		return &LinkError{"rename", oldname, newname, e}
	}
	return nil
}

const chmodMask = uint32(syscall.S_ISUID | syscall.S_ISGID | syscall.S_ISVTX | ModePerm)

// Chmod changes the mode of the named file to mode.
// If the file is a symbolic link, it changes the mode of the link's target.
// If there is an error, it will be of type *PathError.
func Chmod(name string, mode FileMode) error {
	var d syscall.Dir

	d.Null()
	d.Mode = syscallMode(mode) & chmodMask

	var buf [syscall.STATFIXLEN]byte
	n, err := d.Marshal(buf[:])
	if err != nil {
		return &PathError{"chmod", name, err}
	}
	if err = syscall.Wstat(name, buf[:n], syscall.WSTAT_MODE); err != nil {
		return &PathError{"chmod", name, err}
	}
	return nil
}

// Chmod changes the mode of the file to mode.
// If there is an error, it will be of type *PathError.
func (f *File) Chmod(mode FileMode) error {
	if f == nil {
		return ErrInvalid
	}
	var d syscall.Dir

	d.Null()
	d.Mode = syscallMode(mode) & chmodMask

	var buf [syscall.STATFIXLEN]byte
	n, err := d.Marshal(buf[:])
	if err != nil {
		return &PathError{"chmod", f.name, err}
	}
	if err = syscall.Fwstat(f.fd, buf[:n], syscall.WSTAT_MODE); err != nil {
		return &PathError{"chmod", f.name, err}
	}
	return nil
}

// Sync commits the current contents of the file to stable storage.
// Typically, this means flushing the file system's in-memory copy
// of recently written data to disk.
func (f *File) Sync() (err error) {
	if f == nil {
		return ErrInvalid
	}
	if e := syscall.Fsync(f.fd); e != nil {
		return NewSyscallError("fsync", e)
	}
	return nil
}

// Chown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link's target.
// If there is an error, it will be of type *PathError.
func Chown(name string, uid, gid int) error {
	if e := syscall.Chown(name, uid, gid); e != nil {
		return &PathError{"chown", name, e}
	}
	return nil
}

// Lchown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link itself.
// If there is an error, it will be of type *PathError.
func Lchown(name string, uid, gid int) error {
	if e := syscall.Lchown(name, uid, gid); e != nil {
		return &PathError{"lchown", name, e}
	}
	return nil
}

// Chown changes the numeric uid and gid of the named file.
// If there is an error, it will be of type *PathError.
func (f *File) Chown(uid, gid int) error {
	if f == nil {
		return ErrInvalid
	}
	if e := syscall.Fchown(f.fd, uid, gid); e != nil {
		return &PathError{"chown", f.name, e}
	}
	return nil
}

// Chtimes changes the access and modification times of the named
// file, similar to the Unix utime() or utimes() functions.
//
// The underlying filesystem may truncate or round the values to a
// less precise time unit.
// If there is an error, it will be of type *PathError.
func Chtimes(name string, atime time.Time, mtime time.Time) error {
	var d syscall.Dir

	d.Null()
	d.Atime = uint32(atime.Unix())
	d.Mtime = uint32(mtime.Unix())

	var buf [syscall.STATFIXLEN]byte
	n, err := d.Marshal(buf[:])
	if err != nil {
		return &PathError{"chtimes", name, err}
	}
	if err = syscall.Wstat(name, buf[:n], syscall.WSTAT_MTIME|syscall.WSTAT_ATIME); err != nil {
		return &PathError{"chtimes", name, err}
	}
	return nil
}

// basename removes trailing slashes and the leading directory name from path name
func basename(name string) string {
	i := len(name) - 1
	// Remove trailing slashes
	for ; i > 0 && name[i] == '/'; i-- {
		name = name[:i]
	}
	// Remove leading directory name
	for i--; i >= 0; i-- {
		if name[i] == '/' {
			name = name[i+1:]
			break
		}
	}

	return name
}

// TempDir returns the default directory to use for temporary files.
func TempDir() string {
	dir := Getenv("TMPDIR")
	if dir == "" {
		dir = "/tmp"
	}
	return dir
}

func (file *File) AbortOutstandingSyscalls() {
	for file.file.iocount > 0 {
		syscall.AbortSyscFd(file.file.fd)
		time.Sleep(100 * time.Millisecond)
	}
}
