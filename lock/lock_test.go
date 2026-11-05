package lock

import (
	"backupWithRclone/tools"
	"sync"
	"testing"
)

const cTestBackupDir = "../testBackupDir"

func TestLocking(t *testing.T) {
	acquired := make(chan bool)
	attempted := make(chan struct{})
	released := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	// First goroutine: acquire, wait for the competing attempt, release.
	go func() {
		defer wg.Done()

		ok := AcquireLock()
		acquired <- ok

		if !ok {
			t.Error("first goroutine: initial acquisition failed")
			return
		}

		<-attempted
		tools.RunDeferredCalls(false)
		close(released)
	}()

	// Second goroutine: fail while held, succeed after release.
	go func() {
		defer wg.Done()

		if !<-acquired {
			return // Initial acquisition failed; cannot test contention.
		}

		ok := AcquireLock()
		if ok {
			t.Error("second goroutine: acquired lock while already held")
		}
		close(attempted)

		<-released
		ok = AcquireLock()
		if !ok {
			t.Error("second goroutine: acquisition failed after release")
			return
		}
		tools.RunDeferredCalls(false)
	}()

	wg.Wait()
}
