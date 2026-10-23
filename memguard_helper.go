package main

import (
	"github.com/awnumar/memguard"
)

// lockBytes copies b into a memguard-locked buffer and zeroes the original.
// The caller must call Destroy on the returned buffer when done.
func lockBytes(b []byte) (*memguard.LockedBuffer, []byte) {
	if b == nil {
		return nil, nil
	}
	lb := memguard.NewBufferFromBytes(b)
	secureZero(b)
	return lb, lb.Bytes()
}

func destroyLocked(lb *memguard.LockedBuffer) {
	if lb != nil {
		lb.Destroy()
	}
}
