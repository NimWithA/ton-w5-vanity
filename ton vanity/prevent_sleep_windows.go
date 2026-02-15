package main

import "golang.org/x/sys/windows"

func preventSleep() {
	modkernel32 := windows.NewLazySystemDLL("kernel32.dll")
	procSetThreadExecutionState := modkernel32.NewProc("SetThreadExecutionState")

	const (
		ES_CONTINUOUS       = 0x80000000
		ES_SYSTEM_REQUIRED  = 0x00000001
		ES_DISPLAY_REQUIRED = 0x00000002
	)

	procSetThreadExecutionState.Call(
		uintptr(ES_CONTINUOUS | ES_SYSTEM_REQUIRED | ES_DISPLAY_REQUIRED),
	)
}