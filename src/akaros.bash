#!/usr/bin/env bash

# Set things up to build for akaros
export GOOS=akaros
export GOARCH=amd64
export CC_FOR_TARGET=x86_64-ucb-akaros-gcc
export CXX_FOR_TARGET=x86_64-ucb-akaros-g++
export GO_EXTLINK_ENABLED=1
export CGO_ENABLED=1

pre_target_build()
{
	export CGO_ENABLED=1

	# Regenerate all of the files needed by the runtime package
	#
	# The host bootstrap tool needs defs_akaros_amd64.h to exist for its
	# "dist boostrap" command (before Building packages and commands for
	# host).  If the file is empty, some part of our runtime build fails -
	# even if this code autogenerates the defs file first.  Perhaps the
	# file's contents affect something else in the pipeline *before*
	# pre_target_build, which affects the compilation afterwards.
	#
	# Regardless, we'll autogenerate it again.  Any changes to
	# defs_akaros.go will change defs_akaros_amd64.h, which we'll catch
	# with git.
	local ROSINC=$($CC_FOR_TARGET --print-sysroot)/usr/include
	cd runtime
	cp $ROSINC/ros/bits/syscall.h zsyscall_${GOOS}.h
	$GOTOOLDIR/go_bootstrap tool cgo -cdefs defs_${GOOS}.go defsbogus_${GOOS}.go > defs_${GOOS}_${GOARCH}.h
	$GOTOOLDIR/go_bootstrap tool cgo -godefs defs_${GOOS}.go > parlib/zdefs_${GOOS}_${GOARCH}.go
	rm -rf _obj
	cd - > /dev/null

	# Regenerate all of the files needed by the syscall package
	cd syscall
	./mkall.sh > /dev/null 2>&1 
	cd - > /dev/null
}
post_target_build()
{
	unset CGO_ENABLED
}

export -f pre_target_build
export -f post_target_build

# Run the appropriate bash script
targets="clean make run all"
for t in $targets; do
	if [ "$t" = "$1" ]; then 
		export PATH=$(pwd)/../misc/akaros:$PATH
		./$1.bash ${@:2}
		exit $?
	fi
done
echo The first argument to $0 must be one of \{${targets// /, }\}


