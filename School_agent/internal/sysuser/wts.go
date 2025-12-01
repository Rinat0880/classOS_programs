package sysuser

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	wtsapi32 = windows.NewLazySystemDLL("wtsapi32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procWTSGetActiveConsoleSessionId = kernel32.NewProc("WTSGetActiveConsoleSessionId")
	procWTSQuerySessionInformationW  = wtsapi32.NewProc("WTSQuerySessionInformationW")
	procWTSFreeMemory                = wtsapi32.NewProc("WTSFreeMemory")
)

type WTSInfoClass uint32

const (
	WTSUserName     WTSInfoClass = 5
	WTSDomainName   WTSInfoClass = 7
	WTSConnectState WTSInfoClass = 8
)

// GetActiveUser возвращает "DOMAIN\User" или просто "User" того, кто сейчас залогинен физически
func GetActiveUser() (string, error) {
	// 1. Получаем ID сессии, которая сейчас на экране (Active Console)
	r0, _, _ := syscall.SyscallN(procWTSGetActiveConsoleSessionId.Addr())
	sessionID := uint32(r0)
	if sessionID == 0xFFFFFFFF {
		return "", fmt.Errorf("no active console session")
	}

	// 2. Получаем имя пользователя для этой сессии
	user, err := querySessionInfo(uint32(sessionID), WTSUserName)
	if err != nil {
		return "", err
	}

	// 3. (Опционально) Получаем домен
	domain, err := querySessionInfo(uint32(sessionID), WTSDomainName)
	if err == nil && len(domain) > 0 {
		return fmt.Sprintf("%s\\%s", domain, user), nil
	}

	return user, nil
}

func querySessionInfo(sessionID uint32, infoClass WTSInfoClass) (string, error) {
	var buffer *uint16
	var bytesReturned uint32

	// WTS_CURRENT_SERVER_HANDLE = 0
	ret, _, err := procWTSQuerySessionInformationW.Call(
		0,
		uintptr(sessionID),
		uintptr(infoClass),
		uintptr(unsafe.Pointer(&buffer)),
		uintptr(unsafe.Pointer(&bytesReturned)),
	)

	if ret == 0 {
		return "", fmt.Errorf("WTSQuerySessionInformationW failed: %v", err)
	}
	defer procWTSFreeMemory.Call(uintptr(unsafe.Pointer(buffer)))

	return windows.UTF16PtrToString(buffer), nil
}
