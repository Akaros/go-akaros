#!/usr/bin/env bash

# Set things up to build for akaros
export GOOS=akaros
export GOARCH=amd64
export CC_FOR_TARGET=x86_64-ucb-akaros-gcc
export CXX_FOR_TARGET=x86_64-ucb-akaros-g++
export GO_EXTLINK_ENABLED=1
export CGO_ENABLED=0

pre_host_build()
{
	# The host bootstrap tool needs this file to exist, even though we don't
	# need it right away and autogenerate it later on.
	touch pkg/runtime/defs_${GOOS}_${GOARCH}.h
}
pre_target_build()
{
	export CGO_ENABLED=1

	# Regenerate all of the files needed by the runtime package
	local ROSINC=$($CC_FOR_TARGET --print-sysroot)/usr/include
	cd pkg/runtime
	cp $ROSINC/ros/bits/syscall.h zsyscall_${GOOS}.h
	$GOTOOLDIR/go_bootstrap tool cgo -cdefs defs_${GOOS}.go defsbogus_${GOOS}.go > defs_${GOOS}_${GOARCH}.h
	$GOTOOLDIR/go_bootstrap tool cgo -godefs defs_${GOOS}.go > parlib/zdefs_${GOOS}_${GOARCH}.go
	rm -rf _obj
	cd - > /dev/null

	# Regenerate all of the files needed by the syscall package
	cd pkg/syscall
	./mkall.sh > /dev/null 2>&1 
	cd - > /dev/null
}
post_target_build()
{
	unset CGO_ENABLED
}

export -f pre_host_build
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


