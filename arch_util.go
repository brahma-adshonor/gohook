package hook

import (
	"bytes"
	"golang.org/x/arch/x86/x86asm"
	"math"
	"reflect"
	"syscall"
)

type CodeFix struct {
	Code []byte
	Addr uintptr
}

var (
	elfInfo, _     = NewElfInfo()
	funcPrologue32 = []byte{0x64, 0x48, 0x8b, 0x0c, 0x25, 0xf8, 0xff, 0xff, 0xff, 0x48, 0x8d, 0x44, 0x24, 0xe0} //FIXME
	funcPrologue64 = []byte{0x64, 0x48, 0x8b, 0x0c, 0x25, 0xf8, 0xff, 0xff, 0xff, 0x48, 0x8d, 0x44, 0x24, 0xe0}

	// ======================condition jump instruction========================
	// JA JAE JB JBE JCXZ JE JECXZ JG JGE JL JLE JMP JNE JNO JNP JNS JO JP JRCXZ JS

	// one byte opcode, one byte relative offset
	twoByteCondJmp = []byte{0x70, 0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77, 0x78, 0x79, 0x7a, 0x7b, 0x7c, 0x7d, 0x7e, 0x7f, 0xe3}
	// two byte opcode, four byte relative offset
	sixByteCondJmp = []byte{0x0f80, 0x0f81, 0x0f82, 0x0f83, 0x0f84, 0x0f85, 0x0f86, 0x0f87, 0x0f88, 0x0f89, 0x0f8a, 0x0f8b, 0x0f8c, 0x0f8d, 0x0f8e, 0x0f8f}

	// ====================== jump instruction========================
	// one byte opcode, one byte relative offset
	twoByteJmp = []byte{0xeb}
	// one byte opcode, four byte relative offset
	fiveByteJmp = []byte{0xe9}

	// ====================== call instruction========================
	// one byte opcode, 4 byte relative offset
	fiveByteCall = []byte{0xe8}

	// ====================== ret instruction========================
	// return instruction, no operand
	oneByteRet = []byte{0xc3, 0xcb}
	// return instruction, one byte opcode, 2 byte operand
	threeByteRet = []byte{0xc2, 0xca}
)

const (
	FT_CondJmp = 1
	FT_JMP     = 2
	FT_CALL    = 3
	FT_RET     = 4
	FT_OTHER   = 5
	FT_INVALID = 6
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

func FixOneInstruction(mode int, startAddr, curAddr uintptr, code []byte, to uintptr, to_sz int) (int, int, []byte) {
    nc := make([]byte, len(code))
    copy(nc, code)

	if code[0] == 0xe3 || (code[0] >= 0x70 && code[0] <= 0x7f) {
		// two byte condition jump
        newAddr := curAddr
		absAddr := curAddr + 2 + int8(code[1])

        if curAddr < startAddr + to_sz {
            newAddr = to + int(curAddr - startAddr)
        }

        if absAddr >= startAddr && absAddr < startAddr + to_sz {
            absAddr = to + int(absAddr - startAddr)
        }

        nc[1] = byte(absAddr - newAddr - 2)

		return 2, FT_CondJmp, nc
	}

	if code[0] == 0x0f && (code[1] >= 0x80 && code[1] <= 0x8f) {
		// six byte condition jump
		// TODO
		return 6, FT_CondJmp
	}

	if code[0] == 0xeb {
		// two byte jmp
		// TODO
		return 2, FT_JMP
	}

	if code[0] == 0xe9 {
		// five byte jmp
		// TODO
		return 5, FT_JMP
	}

	if code[0] == 0xe8 {
		// five byte call
		// TODO
		return 5, FT_CALL
	}

	// ret instruction just return, no fix is needed.
	if code[0] == 0xc3 || code[0] == 0xcb {
		// one byte ret
		return 1, FT_RET
	}

	if code[0] == 0xc2 || code[0] == 0xca {
		// three byte ret
		return 3, FT_RET
	}

	inst, err := x86asm.Decode(code, mode)
	if err != nil || (inst.Opcode == 0 && inst.Len == 1 && inst.Prefix[0] == x86asm.Prefix(d[0])) {
		return 0, FT_INVALID
	}

	len := inst.Len()
	return len, FT_OTHER
}

// FixTargetFuncCode fix function code starting at address [start]
// parameter 'funcSz' may not specify, in which case, we need to find out the end by scanning next prologue or finding invalid instruction.
// 'to' specifys a new location, to which 'move_sz' bytes instruction will be copied
// since move_sz byte instructions will be copied, those relative jump instruction need to be fixed.
func FixTargetFuncCode(mode int, start uintptr, funcSz int, to uintptr, move_sz int) ([]CodeFix, error) {
	funcPrologue := funcPrologue64
	if mode == 32 {
		funcPrologue = funcPrologue32
	}

	prologueLen := len(funcPrologue)
	code := makeSliceFromPointer(addr, 16) // instruction takes at most 16 bytes

	fix := make([]CodeFix, 0, 64)

	// don't use bytes.Index() as addr may be the last function, which not followed by another function.
	// thus will never find next prologue

	if !bytes.Equal(funcPrologue, code[:prologueLen]) { // not valid function start or invalid prologue
		return nil, errors.New("invalid func prologue")
	}

	curSz := 0
	curAddr := addr + curSz

	for {
		if curSz >= move_sz {
			break
		}

		code = makeSliceFromPointer(curAddr, 16) // instruction takes at most 16 bytes
		sz, ft, nc := FixOneInstruction(mode, addr, curAddr, code, to, move_sz)
		if sz == 0 && ft == FT_INVALID {
			// the end or unrecognized instruction
			return nil, errors.New("ivalid instruction scanned")
		}

		if ft == FT_RET {
			return nil, errors.New("ret instruction in patching erea is not allowed")
		}

		fix = append(fix, CodeFix{Code: nc, Addr: curAddr})

		curSz += sz
		curAddr = addr + curSz
	}

	for {
		if funcSz > 0 && curAddr >= funcSz {
			break
		}

		code = makeSliceFromPointer(curAddr, 16) // instruction takes at most 16 bytes
		if bytes.Equal(funcPrologue, code[:prologueLen]) {
			break
		}

		sz, ft, nc := FixOneInstruction(mode, addr, curAddr, code, to, move_sz)
		if sz == 0 && ft == FT_INVALID {
			// the end or unrecognized instruction
			break
		}

		fix = append(fix, CodeFix{Code: nc, Addr: curAddr})

		curSz += sz
		curAddr = addr + curSz
	}

	return fix, nil
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
