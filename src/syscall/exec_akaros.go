// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// StartProcess and Exec

package syscall

import (
	"unsafe"
	"usys"
)

// ProcAttr holds attributes that will be applied to a new process started
// by StartProcess.
type ProcAttr struct {
	Dir   string       // Current working directory.
	Env   []string     // Environment.
	Files []uintptr    // File descriptors.
	Sys   *SysProcAttr // System specific attrs
}

var zeroProcAttr ProcAttr

// Undefined on Akaros
type SysProcAttr struct{}

var zeroSysProcAttr SysProcAttr

// SlicePtrFromStrings converts a slice of strings to a slice of
// pointers to NUL-terminated byte slices. If any string contains
// a NUL byte, it returns (nil, EINVAL).
func SlicePtrFromStrings(ss []string) ([]*byte, error) {
	var err error
	bb := make([]*byte, len(ss)+1)
	for i := 0; i < len(ss); i++ {
		bb[i], err = BytePtrFromString(ss[i])
		if err != nil {
			return nil, err
		}
	}
	bb[len(ss)] = nil
	return bb, nil
}

// This is allocated in C, and must stay in sync with parlib/serialize.h.  Buf
// is an open-ended C array.
type SerializedData struct {
	Len uintptr
	Buf [1]byte
}

func SerializeArgvEnvp(argv []*byte, envp []*byte) (sd *SerializedData, err error) {
	p_argv := (unsafe.Pointer(&argv[0]))
	p_envp := (unsafe.Pointer(&envp[0]))

	__sd := usys.Call(usys.USYS_SERIALIZE_ARGV_ENVP, uintptr(p_argv), uintptr(p_envp))
	if __sd == 0 {
		err = NewAkaError(1, "SerializeArgvEnvp: error packing argv and envp")
	} else {
		sd = (*SerializedData)(unsafe.Pointer(uintptr(__sd)))
	}
	return sd, err
}

func getSDBuffer(sd *SerializedData) uintptr {
	return uintptr(unsafe.Pointer(&sd.Buf[0]))
}

func FreeSerializedData(sd *SerializedData) {
	usys.Call1(usys.USYS_FREE, uintptr(unsafe.Pointer(sd)))
}

func StartProcess(argv0 string, argv []string, attr *ProcAttr) (pid int, handle uintptr, err error) {
	if attr == nil {
		attr = &zeroProcAttr
	}
	sys := attr.Sys
	if sys == nil {
		sys = &zeroSysProcAttr
	}

	// Convert args to C form.
	argv0p, err := ByteSliceFromString(argv0)
	if err != nil {
		return 0, 0, err
	}
	argvp, err := SlicePtrFromStrings(argv)
	if err != nil {
		return 0, 0, err
	}
	envvp, err := SlicePtrFromStrings(attr.Env)
	if err != nil {
		return 0, 0, err
	}

	// Kick off child.
	pid, err = startProcess(argv0p, argvp, envvp, attr.Dir, attr.Files)

	// Return the pid and the error if there was one
	return pid, 0, err
}

func startProcess(argv0 []byte, argv, envv []*byte, dir string, files []uintptr) (pid int, err error) {
	// Adjust argv0 to prepend 'dir' if argv0 is a relative path
	if argv0[0] != '/' {
		if len(dir) > 0 {
			argv0 = append([]byte(dir[:len(dir)-1]),
			               append([]byte{'/'}, argv0...)...)
		}
	}

	sd, err := SerializeArgvEnvp(argv, envv)
	if err != nil {
		return 0, err
	}
	// sd was allocated in C, so it's not a Go object/pointer.  We don't
	// need to worry about stack splits or garbage collection.
	child, err := ProcCreate(argv0, getSDBuffer(sd), sd.Len, 0)
	FreeSerializedData(sd)
	if err != nil {
		return 0, err
	}

	__cfdm := make([]Childfdmap_t, 0, len(files))
	for i, f := range files {
		if int(f) < 0 {
			continue
		}
		__cfdm = append(__cfdm, Childfdmap_t{Parentfd: uint32(f),
		                                     Childfd: uint32(i),
		                                     Ok: int32(-1)})
	}

	// We're relying on the slice internals; that the contents are an array
	// of objects.
	_, err = DupFdsTo(child, &__cfdm[0], len(__cfdm))
	if err != nil {
		return 0, err
	}

	if len(dir) > 0 {
		err = chdir(child, dir)
		if err != nil {
			return 0, err
		}
	}

	err = ProcRun(child)
	if err != nil {
		return 0, err
	}

	return child, nil
}

// Ordinary exec.
func Exec(argv0 string, argv []string, envv []string) (err error) {
	// Convert args to C form.
	argv0p, err := ByteSliceFromString(argv0)
	if err != nil {
		return err
	}
	argvp, err := SlicePtrFromStrings(argv)
	if err != nil {
		return err
	}
	envvp, err := SlicePtrFromStrings(envv)
	if err != nil {
		return err
	}
	sd, err := SerializeArgvEnvp(argvp, envvp)
	if err != nil {
		return err
	}
	err = exec(argv0p, getSDBuffer(sd), sd.Len)
	FreeSerializedData(sd)
	return err
}

func CloseOnExec(fd int) { fcntl(fd, F_SETFD, FD_CLOEXEC) }
