package gohook

import (
	"syscall"
	"unsafe"
)

const PAGE_EXECUTE_READWRITE = 0x40

var procVirtualProtect = syscall.NewLazyDLL("kernel32.dll").NewProc("VirtualProtect")

func virtualProtect(lpAddress uintptr, dwSize int, flNewProtect uint32, lpflOldProtect unsafe.Pointer) error {
	ret, _, _ := procVirtualProtect.Call(
		lpAddress,
		uintptr(dwSize),
		uintptr(flNewProtect),
		uintptr(lpflOldProtect))
	if ret == 0 {
		return syscall.GetLastError()
	}
	return nil
}

func CopyInstruction(location uintptr, data []byte) {
	f := makeSliceFromPointer(location, len(data))

	var oldPerms uint32
	err := virtualProtect(location, len(data), PAGE_EXECUTE_READWRITE, unsafe.Pointer(&oldPerms))
	if err != nil {
		panic(err)
	}

	copy(f, data[:])

	var tmp uint32
	err = virtualProtect(location, len(data), oldPerms, unsafe.Pointer(&tmp))
	if err != nil {
		panic(err)
	}
}
