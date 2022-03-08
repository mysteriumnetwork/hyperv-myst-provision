package utils

import "syscall"

func getSysProcAttrs() syscall.SysProcAttr {
	return syscall.SysProcAttr{}
}
