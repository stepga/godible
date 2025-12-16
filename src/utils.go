package godible

import "syscall"

// Reboot syncs the file system cache and performs the default restart.
func Reboot() {
	syscall.Sync()
	// `man 2 reboot`:
	// (RB_AUTOBOOT, 0x1234567).  The message "Restarting system." is
	// printed, and a default restart is performed immediately. If not
	// preceded by a sync(2), data will be lost.
	syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
}

// RemountPerm remounts the hardcoded partition. If the parameter is true,
// the partition will be remounted readonly, otherwirse it will be remounted
// writable.
func RemountPerm(readonly bool) error {
	mountSrc := "/dev/mmcblk0p4"
	mountDst := "/perm"
	fsType := "ext4"
	// relatime: performace optimization; update file access time only when necessary
	var mountFlags uintptr = syscall.MS_REMOUNT | syscall.MS_RELATIME
	mountData := ""
	if readonly {
		mountFlags = syscall.MS_REMOUNT | syscall.MS_RDONLY
	}
	return syscall.Mount(mountSrc, mountDst, fsType, mountFlags, mountData)
}
