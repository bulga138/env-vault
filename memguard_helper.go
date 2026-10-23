package main

import (
	"github.com/awnumar/memguard"
)

func lockPassword(p []byte) (*memguard.LockedBuffer, []byte) {
	if p == nil {
		return nil, nil
	}
	lb := memguard.NewBufferFromBytes(p)
	secureZero(p)
	return lb, lb.Bytes()
}