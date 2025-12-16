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

func RemountPerm(readonly bool) error {
	// relatime: performace optimization; update file access time only when necessary
	var flags uintptr = syscall.MS_REMOUNT | syscall.MS_RELATIME
	if readonly {
		flags = syscall.MS_REMOUNT | syscall.MS_RDONLY
	}

	return syscall.Mount("/dev/mmcblk0p4", "/perm", "ext4", flags, "")
}
