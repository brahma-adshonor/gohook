package hook

import (
	"reflect"
	"syscall"
	"unsafe"
)

func makeSliceFromPointer(p uintptr, length int) []byte {
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: p,
		Len:  length,
		Cap:  length,
	}))
}

func CopyInstruction(location uintptr, data []byte) {
	f := makeSliceFromPointer(location, len(data))
	setPageWritable(location, len(data), syscall.PROT_READ|syscall.PROT_WRITE|syscall.PROT_EXEC)
	copy(f, data[:])
	setPageWritable(location, len(data), syscall.PROT_READ|syscall.PROT_EXEC)
}

func hookFunction(mode int, target, replace, trampoline uintptr) ([]byte, error) {
	jumpcode := genJumpCode(mode, replace, target)

	insLen := len(jumpcode)
	if trampoline != uintptr(0) {
		f := makeSliceFromPointer(target, len(jumpcode)*2)
		insLen = GetInsLenGreaterThan(mode, f, len(jumpcode))
	}

	// target slice
	ts := makeSliceFromPointer(target, insLen)
	original = make([]byte, len(ts))
	copy(original, ts)

	if trampoline != uintptr(0) {
		sz := 0
		if elfInfo != nil {
			sz = elfInfo.GetFuncSize(addr)
		}

		err := FixTargetFuncCode(mode, target, sz, trampoline, insLen)
		if err != nil {
			return nil, err
		}
	}

	CopyInstruction(target, jumpcode)

	if trampoline != uintptr(0) {
		target_end := FindFuncEnd(target)
		CopyInstruction(trampoline, ts)
		jumpcode := genJumpCode(mode, target+uintptr(insLen), trampoline+uintptr(insLen))
		CopyInstruction(trampoline+uintptr(insLen), jumpcode)
	}

	return original, nil
}
