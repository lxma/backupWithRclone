package lock

import (
	"backupWithRclone/config"
	"backupWithRclone/logger"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type Lock struct {
	lockFile *os.File
}

// AcquireInstanceLock returns an lock that shall remain set
// for the entire protected operation.
func AcquireLock() (Lock, bool) {
	lockFileName := filepath.Join(config.GBackupDir, "lock")
	lockFile, err := os.OpenFile(lockFileName, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		logger.Printf("error opening lock file %s: %v\n", lockFileName, err)
		panic(fmt.Errorf("failed to open lock file %s: %v", lockFileName, err))
	}

	err = unix.Flock(int(lockFile.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		lockFile.Close()

		if errors.Is(err, unix.EWOULDBLOCK) ||
			errors.Is(err, unix.EAGAIN) {
			return Lock{}, false
		}
		logger.Printf("unexpected error acquiring instance lock %s: %v\n", lockFileName, err)
		panic(fmt.Errorf("unexpected error acquiring instance lock %s: %v", lockFileName, err))
	}

	// Don't overwrite the gLockFile before the lock could be
	// acquired. Otherwise, unit tests will fail.
	return Lock{lockFile}, true
}

func (l Lock) ReleaseLock() {
	l.lockFile.Close()
}
