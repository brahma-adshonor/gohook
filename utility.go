package hook

import (
	"golang.org/x/arch/x86/x86asm"
	"math"
    "bytes"
	"reflect"
	"syscall"
	"unsafe"
)

var (
	elfInfo, _ = NewElfInfo()
    funcPrologue32 = []byte{0x64,0x48,0x8b,0x0c,0x25,0xf8,0xff,0xff,0xff,0x48,0x8d,0x44,0x24,0xe0}
    funcPrologue64 = []byte{0x64,0x48,0x8b,0x0c,0x25,0xf8,0xff,0xff,0xff,0x48,0x8d,0x44,0x24,0xe0}
)

func SetFuncPrologue(mode int, data []byte) {
    if mode == 32 {
        funcPrologue32 = make([]byte, len(data))
        copy(funcPrologue32, data)
    } else {
        funcPrologue64 = make([]byte, len(data))
        copy(funcPrologue64, data)
    }
}

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
	for p := getPageAddr(addr); p < addr+uintptr(length); p += uintptr(pageSize) {
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

func GetInsLenGreaterThan(mode int, data []byte, least int) int {
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

		inst, err := x86asm.Decode(d, mode)
		if err != nil || (inst.Opcode == 0 && inst.Len == 1 && inst.Prefix[0] == x86asm.Prefix(d[0])) {
			break
		}

		curLen = curLen + inst.Len
		d = data[curLen:]
	}

	return curLen
}


type JumpInstruction struct {
    addr uintptr
    inst x86asm.Inst
}

// ======================condition jump instruction========================
// JA JAE JB JBE JCXZ JE JECXZ JG JGE JL JLE JMP JNE JNO JNP JNS JO JP JRCXZ JS

// one byte opcode, one byte relative offset
twoByteCondJmp := {0x70,0x71,0x72,0x73,0x74,0x75,0x76,0x77,0x78,0x79,0x7a,0x7b,0x7c,0x7d,0x7e,0x7f,0xe3}
// two byte opcode, four byte relative offset
sixByteCondJmp := {0x0f80,0x0f81,0x0f82,0x0f83,0x0f84,0x0f85,0x0f86,0x0f87,0x0f88,0x0f89,0x0f8a,0x0f8b,0x0f8c,0x0f8d,0x0f8e,0x0f8f}


// ====================== jump instruction========================
// one byte opcode, one byte relative offset
twoByteJmp := {0xeb}
// one byte opcode, four byte relative offset
fiveByteJmp := {0xe9}


// ====================== call instruction========================
// one byte opcode, 4 byte relative offset
fiveByteCall := {0xe8}


// ====================== ret instruction========================
// return instruction, no operand
oneByteRet := {0xc3, 0xcb}
// return instruction, one byte opcode, 2 byte operand
threeByteRet := {0xc2, 0xca}


const {
    FT_CondJmp = 1
    FT_JMP = 2
    FT_CALL = 3
    FT_RET = 4
    FT_OTHER = 5
    FT_INVALID = 6
}

func FixOneInstruction(mode int, addr uintptr, code []byte, to uintptr, to_sz int) (int, FixType) {
    if code[0] == 0xe3 || (code[0] >= 0x70 && code[0] <= 0x7f) {
        // two byte condition jump
        // TODO
        return (2, FT_CondJmp)
    }

    if code[0] == 0x0f && (code[1] >= 0x80 && code[1] <= 0x8f) {
        // six byte condition jump
        // TODO
        return (6, FT_CondJmp)
    }

    if code[0] == 0xeb {
        // two byte jmp
        // TODO
        return (2, FT_JMP)
    }

    if code[0] == 0xe9 {
        // five byte jmp
        // TODO
        return (5, FT_JMP)
    }

    if code[0] == 0xe8 {
        // five byte call
        // TODO
        return (5, FT_CALL)
    }

    if code[0] == 0xc3 || code[0] == 0xcb {
        // one byte ret
        // TODO
        return (1, FT_RET)
    }

    if code[0] == 0xc2 || code[0] == 0xca {
        // three byte ret
        // TODO
        return (3, FT_RET)
    }

    inst, err := x86asm.Decode(code, mode)
    if err != nil || (inst.Opcode == 0 && inst.Len == 1 && inst.Prefix[0] == x86asm.Prefix(d[0])) {
        return (0, FT_INVALID)
    }

    len := inst.Len()
    return (len, FT_OTHER)
}

// FixJmpCode fix function code starting at address [start]
// parameter 'end' may not specify, in which case, we need to find out the end by scanning next prologue or finding invalid instruction.
// 'to' specifys a new location, to which 'move_sz' bytes instruction will be copied
// since move_sz byte instructions will be copied, those relative jump instruction need to be fixed.
func FixJmpCode(mode int, start, end uintptr, to uintptr, move_sz int) {
    funcPrologue := funcPrologue64
    if mode == 32 {
        funcPrologue = funcPrologue32
    }

    prologueLen := len(funcPrologue)
    code := makeSliceFromPointer(addr, 16) // instruction takes at most 16 bytes

    // don't use bytes.Index() as addr may be the last function, which not followed by another function.
    // thus will never find next prologue

    if !bytes.Equal(funcPrologue, code[:prologueLen]) { // not valid function start or invalid prologue
        return 0
    }

    sz, ft := FixOneInstruction(mode, addr, code, to, move_sz)

    curAddr := addr + prologueLen

    for {
        code = makeSliceFromPointer(curAddr, 16) // instruction takes at most 16 bytes
        if bytes.Equal(funcPrologue, code[:prologueLen]) {
            return curAddr
        }

		inst, err := x86asm.Decode(code, mode)
		if err != nil || (inst.Opcode == 0 && inst.Len == 1 && inst.Prefix[0] == x86asm.Prefix(d[0])) {
            return curAddr
		}

        curAddr += inst.Len()
    }

    panic("search function prologue failed")
    return 0
}

func FindFuncEnd(addr uintptr) uintptr {
	sz := 0
	if elfInfo != nil {
		sz = elfInfo.GetFuncSize(addr)
	}

	if sz == 0 {
	}

	return addr + sz
}

func TransformInstruction(code []byte, fs, fe uintptr) []byte {
	// TODO: 
    // fix relative jmp instruction.
	// call jmp jne jge etc.
    // ret instruction with code

    if fs >= fe {
        panic("invalid function start/end addr")
    }

	ret := make([]byte, len(code))
	copy(ret, code)

	return ret
}

func genJumpCode(mode int, to, from uintptr) []byte {
	// 1. use relaive jump if |from-to| < 2G
	// 2. otherwise, push target, then ret

	relative := (uint32(math.Abs(float64(from-to))) < 0x7fffffff)
	if relative {
		var dis uint32
		if to > from {
			dis = uint32(int32(to-from) + 5)
		} else {
			dis = uint32(-int32(from-to) - 5)
		}
		return []byte{
			0xe9,
			byte(dis),
			byte(dis >> 8),
			byte(dis >> 16),
			byte(dis >> 24),
		}
	}

	if mode == 32 {
		return []byte{
			0x68, // push
			byte(to),
			byte(to >> 8),
			byte(to >> 16),
			byte(to >> 24),
			0xc3, // retn
		}
	} else if mode == 64 {
		// push does not operate on 64bit imm, workarounds are:
		// 1. move to register(eg, %rdx), then push %rdx, however, overwriting register may cause problem if not handled carefully.
		// 2. push twice, preferred.
		/*
		   return []byte{
		       0x48, // prefix
		       0xba, // mov to %rdx
		       byte(to), byte(to >> 8), byte(to >> 16), byte(to >> 24),
		       byte(to >> 32), byte(to >> 40), byte(to >> 48), byte(to >> 56),
		       0x52, // push %rdx
		       0xc3, // retn
		   }
		*/
		return []byte{
			0x68, //push
			byte(to), byte(to >> 8), byte(to >> 16), byte(to >> 24),
			0xc7, 0x44, 0x24, // mov $value, -4%rsp
			0xfc, // rsp - 4
			byte(to >> 32), byte(to >> 40), byte(to >> 48), byte(to >> 56),
			0xc3, // retn
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
		insLen = GetInsLenGreaterThan(mode, f, len(jumpcode))
	}

	// target slice
	ts := makeSliceFromPointer(target, insLen)
	original = make([]byte, len(ts))

	copy(original, ts)

	CopyInstruction(target, jumpcode)

	if trampoline != uintptr(0) {
		target_end := FindFuncEnd(target)
		code := TransformInstruction(original, target, target_end)
		CopyInstruction(trampoline, code)
		jumpcode := genJumpCode(mode, target+uintptr(insLen), trampoline+uintptr(insLen))
		CopyInstruction(trampoline+uintptr(insLen), jumpcode)
	}

	return
}
