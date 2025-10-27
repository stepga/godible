package godible

import "syscall"

func Reboot() {
	syscall.Sync()
	// `man 2 reboot`:
	// (RB_AUTOBOOT, 0x1234567).  The message "Restarting system." is
	// printed, and a default restart is performed immediately. If not
	// preceded by a sync(2), data will be lost.
	syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
}
