package lock

import (
	"time"

	"6.5840/kvsrv1/rpc"
	kvtest "6.5840/kvtest1"
)

type LockStatus string

const (
	LockStatusFree LockStatus = "free"
	LockStatusHeld LockStatus = "held"
)

type Lock struct {
	// IKVClerk is a go interface for k/v clerks: the interface hides
	// the specific Clerk type of ck but promises that ck supports
	// Put and Get.  The tester passes the clerk in when calling
	// MakeLock().
	ck kvtest.IKVClerk
	// You may add code here
	lockname string
}

// The tester calls MakeLock() and passes in a k/v clerk; your code can
// perform a Put or Get by calling lk.ck.Put() or lk.ck.Get().
//
// This interface supports multiple locks by means of the
// lockname argument; locks with different names should be
// independent.
func MakeLock(ck kvtest.IKVClerk, lockname string) *Lock {
	lk := &Lock{ck: ck, lockname: lockname}
	return lk
}

func (lk *Lock) Acquire() {
	for {
		if val, version, err := lk.ck.Get(lk.lockname); err == rpc.ErrNoKey || (err == rpc.OK && val == string(LockStatusFree)) {
			if err := lk.ck.Put(lk.lockname, string(LockStatusHeld), version); err == rpc.OK {
				return
			}
		}
		time.Sleep(time.Millisecond * 100)
	}
}

func (lk *Lock) Release() {
	for {
		if val, version, err := lk.ck.Get(lk.lockname); err == rpc.ErrNoKey || (err == rpc.OK && val == string(LockStatusHeld)) {
			if err := lk.ck.Put(lk.lockname, string(LockStatusFree), version); err == rpc.OK {
				return
			}
		}
		time.Sleep(time.Millisecond * 100)
	}
}
