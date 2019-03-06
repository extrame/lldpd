// +build !windows

package lldpd

import (
	"golang.org/x/sys/unix"
)

func isShouldFinishError(err error) bool {
	return err == unix.EBADF
}
