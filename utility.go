package hook

import (
	"reflect"
	"syscall"
	"unsafe"
	"math"
	"golang.org/x/arch/x86/x86asm"
)

func makeSliceFromPointer(p uintptr, length int) []byte {
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: p,
		Len:  length,
		Cap:  length,
	}))
}

func getPageAddr(ptr uintptr) uintptr {
	return ptr & ^(uintptr(syscall.Getpagesize() - 1))
}

func setPageWritable(addr uintptr, length int, prot int) {
	pageSize := syscall.Getpagesize()
	for p := getPageAddr(addr); p < addr + uintptr(length); p += uintptr(pageSize) {
		page := makeSliceFromPointer(p, pageSize)
		err := syscall.Mprotect(page, prot)
		if err != nil {
			panic(err)
		}
	}
}

func CopyInstruction(location uintptr, data []byte) {
	f := makeSliceFromPointer(location, len(data))
	setPageWritable(location, len(data), syscall.PROT_READ|syscall.PROT_WRITE|syscall.PROT_EXEC)
	copy(f, data[:])
	setPageWritable(location, len(data), syscall.PROT_READ|syscall.PROT_EXEC)
}

func GetInsLenGreaterThan(data []byte, least int) int {
	if len(data) < least {
		return 0
	}

	curLen := 0
	d := data[curLen:]
	for {
		if len(d) <= 0 {
			break
		}

		if curLen >= least {
			break
		}

		inst, err := x86asm.Decode(d, 64)
		if err != nil || (inst.Opcode == 0 && inst.Len == 1 && inst.Prefix[0] == x86asm.Prefix(d[0])) {
			break
		}

		curLen = curLen + inst.Len
		d = data[curLen:]
	}

	return curLen
}

func TransformInstruction(code []byte) []byte {
	// TODO: fix relative jmp instruction.
	// jmp jne jge etc.
	ret := make([]byte, 0, len(code))
	copy(ret, code)
	return ret
}

func genJumpCode(mode int, to, from uintptr) []byte {
	// 1. use relaive jump if |from-to| < 2G
	// 2. otherwise, push target, then ret

	use_relative := (uint32(math.Abs(float64(from-to))) < 0x7fffffff)
	if (use_relative) {
		return []byte {
			0xe9,
			byte(to),
			byte(to>>8),
			byte(to>>16),
			byte(to>>24),
		}
	}

	if (mode == 32) {
		return []byte {
			0x68, // push
			byte(to),
			byte(to>>8),
			byte(to>>16),
			byte(to>>24),
			0xc3, // retn
		}
	} else if (mode == 64) {
		return []byte {
			0x48, // prefix
			0x68, // push
			byte(to),
			byte(to >> 8),
			byte(to >> 16),
			byte(to >> 24),
			byte(to >> 32),
			byte(to >> 40),
			byte(to >> 48),
			byte(to >> 56),
			0xc3,
		}
	} else {
		panic("invalid mode")
	}
}

func hookFunction(mode int, target, replace, trampoline uintptr) (original []byte) {
	jumpcode := genJumpCode(mode, replace, target)

	insLen := len(jumpcode)
	if trampoline != uintptr(0) {
		f := makeSliceFromPointer(target, len(jumpcode)*2)
		insLen = GetInsLenGreaterThan(f, len(jumpcode))
	}

	// target slice
	ts := makeSliceFromPointer(target, insLen)
	original = make([]byte, len(ts))

	copy(original, ts)

	CopyInstruction(target, jumpcode)

	if trampoline != uintptr(0) {
		code := TransformInstruction(original)
		CopyInstruction(trampoline, code)
		jumpcode := genJumpCode(mode, target + uintptr(insLen), replace + uintptr(insLen))
		CopyInstruction(trampoline+uintptr(insLen), jumpcode)
	}

	return
}

