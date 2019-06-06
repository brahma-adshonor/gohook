package gohook

import (
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/arch/x86/x86asm"
	"math"
	"syscall"
)

type CodeFix struct {
	Code []byte
	Addr uintptr
}

var (
	minJmpCodeSize = 0
	elfInfo, _     = NewElfInfo()
	funcPrologue32 = []byte{0x65, 0x8b, 0x0d, 0x00, 0x00, 0x00, 0x00, 0x8b, 0x89, 0xfc, 0xff, 0xff, 0xff}
	funcPrologue64 = []byte{0x64, 0x48, 0x8b, 0x0c, 0x25, 0xf8, 0xff, 0xff, 0xff, 0x48}

	// ======================condition jump instruction========================
	// JA JAE JB JBE JCXZ JE JECXZ JG JGE JL JLE JMP JNE JNO JNP JNS JO JP JRCXZ JS

	// one byte opcode, one byte relative offset
	twoByteCondJmp = []byte{0x70, 0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77, 0x78, 0x79, 0x7a, 0x7b, 0x7c, 0x7d, 0x7e, 0x7f, 0xe3}
	// two byte opcode, four byte relative offset
	sixByteCondJmp = []uint16{0x0f80, 0x0f81, 0x0f82, 0x0f83, 0x0f84, 0x0f85, 0x0f86, 0x0f87, 0x0f88, 0x0f89, 0x0f8a, 0x0f8b, 0x0f8c, 0x0f8d, 0x0f8e, 0x0f8f}

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
	FT_CondJmp  = 1
	FT_JMP      = 2
	FT_CALL     = 3
	FT_RET      = 4
	FT_OTHER    = 5
	FT_INVALID  = 6
	FT_SKIP     = 7
	FT_OVERFLOW = 8
)

func SetMinJmpCodeSize(sz int) {
	minJmpCodeSize = sz
}

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

		if inst.Len == 1 && d[0] == 0xcc {
			// 0xcc -> int3, trap to debugger, padding to function end
			break
		}

		curLen = curLen + inst.Len
		d = data[curLen:]
	}

	return curLen
}

func isByteOverflow(v int32) bool {
	if v > 0 {
		if v > math.MaxInt8 {
			return true
		}
	} else {
		if v < math.MinInt8 {
			return true
		}
	}

	return false
}

func isIntOverflow(v int64) bool {
	if v > 0 {
		if v > math.MaxInt32 {
			return true
		}
	} else {
		if v < math.MinInt32 {
			return true
		}
	}

	return false
}

func calcOffset(insSz int, startAddr, curAddr, to uintptr, to_sz int, offset int32) int64 {
	newAddr := curAddr
	absAddr := curAddr + uintptr(insSz) + uintptr(offset)

	if curAddr < startAddr+uintptr(to_sz) {
		newAddr = to + (curAddr - startAddr)
	}

	if absAddr >= startAddr && absAddr < startAddr+uintptr(to_sz) {
		absAddr = to + (absAddr - startAddr)
	}

	return int64(uint64(absAddr) - uint64(newAddr) - uint64(insSz))
}

func FixOneInstruction(mode int, fix_recursive_call bool, startAddr, curAddr uintptr, code []byte, to uintptr, to_sz int) (int, int, []byte) {
	nc := make([]byte, len(code))
	copy(nc, code)

	if code[0] == 0xe3 || code[0] == 0xeb || (code[0] >= 0x70 && code[0] <= 0x7f) {
		// two byte condition jump, two byte jmp
		nc = nc[:2]
		off := calcOffset(2, startAddr, curAddr, to, to_sz, int32(int8(code[1])))
		if off != int64(int8(nc[1])) {
			if isByteOverflow(int32(off)) {
				// overfloat, cannot fix this with one byte operand
				return 2, FT_OVERFLOW, nc
			}
			nc[1] = byte(off)
			return 2, FT_CondJmp, nc
		}
		return 2, FT_SKIP, nc
	}

	if code[0] == 0x0f && (code[1] >= 0x80 && code[1] <= 0x8f) {
		// six byte condition jump
		nc = nc[:6]
		off1 := (uint32(code[2]) | (uint32(code[3]) << 8) | (uint32(code[4]) << 16) | (uint32(code[5]) << 24))
		off2 := uint64(calcOffset(6, startAddr, curAddr, to, to_sz, int32(off1)))
		if uint64(int32(off1)) != off2 {
			if isIntOverflow(int64(off2)) {
				// overfloat, cannot fix this with four byte operand
				return 6, FT_OVERFLOW, nc
			}
			nc[2] = byte(off2)
			nc[3] = byte(off2 >> 8)
			nc[4] = byte(off2 >> 16)
			nc[5] = byte(off2 >> 24)
			return 6, FT_CondJmp, nc
		}
		return 6, FT_SKIP, nc
	}

	if code[0] == 0xe9 || code[0] == 0xe8 {
		// five byte jmp, five byte call
		nc = nc[:5]
		off1 := (uint32(code[1]) | (uint32(code[2]) << 8) | (uint32(code[3]) << 16) | (uint32(code[4]) << 24))

		off2 := uint64(0)
		if !fix_recursive_call && code[0] == 0xe8 && startAddr == (curAddr+uintptr(5)+uintptr(int32(off1))) {
			// don't fix recursive call
			off2 = uint64(int32(off1))
		} else {
			off2 = uint64(calcOffset(5, startAddr, curAddr, to, to_sz, int32(off1)))
		}

		if uint64(int32(off1)) != off2 {
			if isIntOverflow(int64(off2)) {
				// overfloat, cannot fix this with four byte operand
				return 5, FT_OVERFLOW, nc
			}
			nc[1] = byte(off2)
			nc[2] = byte(off2 >> 8)
			nc[3] = byte(off2 >> 16)
			nc[4] = byte(off2 >> 24)
			return 5, FT_JMP, nc
		}
		return 5, FT_SKIP, nc
	}

	// ret instruction just return, no fix is needed.
	if code[0] == 0xc3 || code[0] == 0xcb {
		// one byte ret
		nc = nc[:1]
		return 1, FT_RET, nc
	}

	if code[0] == 0xc2 || code[0] == 0xca {
		// three byte ret
		nc = nc[:3]
		return 3, FT_RET, nc
	}

	inst, err := x86asm.Decode(code, mode)
	if err != nil || (inst.Opcode == 0 && inst.Len == 1 && inst.Prefix[0] == x86asm.Prefix(code[0])) {
		return 0, FT_INVALID, nc
	}

	if inst.Len == 1 && code[0] == 0xcc {
		return 0, FT_INVALID, nc
	}

	sz := inst.Len
	nc = nc[:sz]
	return sz, FT_OTHER, nc
}

// FixTargetFuncCode fix function code starting at address [start]
// parameter 'funcSz' may not specify, in which case, we need to find out the end by scanning next prologue or finding invalid instruction.
// 'to' specifys a new location, to which 'move_sz' bytes instruction will be copied
// since move_sz byte instructions will be copied, those relative jump instruction need to be fixed.
func FixTargetFuncCode(mode int, start uintptr, funcSz uint32, to uintptr, move_sz int) ([]CodeFix, error) {
	funcPrologue := funcPrologue64
	if mode == 32 {
		funcPrologue = funcPrologue32
	}

	prologueLen := len(funcPrologue)
	code := makeSliceFromPointer(start, 16) // instruction takes at most 16 bytes

	fix := make([]CodeFix, 0, 64)

	// don't use bytes.Index() as 'start' may be the last function, which not followed by another function.
	// thus will never find next prologue

	if funcSz == 0 && !bytes.Equal(funcPrologue, code[:prologueLen]) { // not valid function start or invalid prologue
		return nil, errors.New(fmt.Sprintf("invalid func prologue, addr:0x%x", start))
	}

	curSz := 0
	curAddr := start

	for {
		if curSz >= move_sz {
			break
		}

		code = makeSliceFromPointer(curAddr, 16) // instruction takes at most 16 bytes
		sz, ft, nc := FixOneInstruction(mode, false, start, curAddr, code, to, move_sz)
		if sz == 0 && ft == FT_INVALID {
			// the end or unrecognized instruction
			return nil, errors.New(fmt.Sprintf("invalid instruction scanned, addr:0x%x", curAddr))
		}

		if ft == FT_RET {
			return nil, errors.New(fmt.Sprintf("ret instruction in patching erea is not allowed, addr:0x%x", curAddr))
		}

		if ft == FT_OVERFLOW {
			return nil, errors.New(fmt.Sprintf("jmp instruction in patching erea overflow, addr:0x%x", curAddr))
		}

		if ft != FT_OTHER && ft != FT_SKIP {
			fix = append(fix, CodeFix{Code: nc, Addr: curAddr})
		}

		curSz += sz
		curAddr = start + uintptr(curSz)
	}

	for {
		if funcSz > 0 && uint32(curAddr-start) >= funcSz {
			break
		}

		code = makeSliceFromPointer(curAddr, 16) // instruction takes at most 16 bytes
		if funcSz == 0 && bytes.Equal(funcPrologue, code[:prologueLen]) {
			break
		}

		sz, ft, nc := FixOneInstruction(mode, false, start, curAddr, code, to, move_sz)
		if sz == 0 && ft == FT_INVALID {
			// the end or unrecognized instruction
			break
		}

		if ft == FT_OVERFLOW {
			return nil, errors.New(fmt.Sprintf("jmp instruction in body overflow, addr:0x%x", curAddr))
		}

		if ft != FT_OTHER && ft != FT_RET && ft != FT_SKIP {
			fix = append(fix, CodeFix{Code: nc, Addr: curAddr})
		}

		curSz += sz
		curAddr = start + uintptr(curSz)
	}

	return fix, nil
}

func GetFuncSizeByGuess(mode int, start uintptr, minimal bool) (uint32, error) {
	funcPrologue := funcPrologue64
	if mode == 32 {
		funcPrologue = funcPrologue32
	}

	prologueLen := len(funcPrologue)
	code := makeSliceFromPointer(start, 16) // instruction takes at most 16 bytes

	if !bytes.Equal(funcPrologue, code[:prologueLen]) { // not valid function start or invalid prologue
		return 0, errors.New(fmt.Sprintf("no func prologue, addr:0x%x", start))
	}

	int3_found := false
	curLen := uint32(0)

	for {
		inst, err := x86asm.Decode(code, mode)
		if err != nil || (inst.Opcode == 0 && inst.Len == 1 && inst.Prefix[0] == x86asm.Prefix(code[0])) {
			return curLen, nil
		}

		if inst.Len == 1 && code[0] == 0xcc {
			// 0xcc -> int3, trap to debugger, padding to function end
			if minimal {
				return curLen, nil
			}
			int3_found = true
		} else if int3_found {
			return curLen, nil
		}

		curLen = curLen + uint32(inst.Len)
		code = makeSliceFromPointer(start+uintptr(curLen), 16) // instruction takes at most 16 bytes

		if bytes.Equal(funcPrologue, code[:prologueLen]) {
			return curLen, nil
		}
	}

	return 0, nil
}

func copyFuncInstruction(mode int, from, to uintptr, sz int) ([]CodeFix, error) {
	curSz := 0
	curAddr := from
	fix := make([]CodeFix, 0, 256)

	for {
		if curSz >= sz {
			break
		}

		code := makeSliceFromPointer(curAddr, 16) // instruction takes at most 16 bytes
		sz, ft, nc := FixOneInstruction(mode, true, from, curAddr, code, to, sz)

		if sz == 0 && ft == FT_INVALID {
			// the end or unrecognized instruction
			break
		}

		if ft == FT_OVERFLOW {
			return nil, errors.New(fmt.Sprintf("overflow instruction in copying function, addr:0x%x", curAddr))
		}

		to_addr := (to + (curAddr - from))
		fix = append(fix, CodeFix{Code: nc, Addr: to_addr})

		curSz += sz
		curAddr = from + uintptr(curSz)
	}

	return fix, nil
}

func genJumpCode(mode int, to, from uintptr) []byte {
	// 1. use relaive jump if |from-to| < 2G
	// 2. otherwise, push target, then ret

	var code []byte
	relative := (uint32(math.Abs(float64(from-to))) < 0x7fffffff)

	if relative {
		var dis uint32
		if to > from {
			dis = uint32(int32(to-from) - 5)
		} else {
			dis = uint32(-int32(from-to) - 5)
		}
		code = []byte{
			0xe9,
			byte(dis),
			byte(dis >> 8),
			byte(dis >> 16),
			byte(dis >> 24),
		}
	} else if mode == 32 {
		code = []byte{
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
		   code = []byte{
		       0x48, // prefix
		       0xba, // mov to %rdx
		       byte(to), byte(to >> 8), byte(to >> 16), byte(to >> 24),
		       byte(to >> 32), byte(to >> 40), byte(to >> 48), byte(to >> 56),
		       0x52, // push %rdx
		       0xc3, // retn
		   }
		*/
		code = []byte{
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

	sz := len(code)
	if minJmpCodeSize > 0 && sz < minJmpCodeSize {
		nop := make([]byte, 0, minJmpCodeSize-sz)
		for {
			if len(nop) >= minJmpCodeSize-sz {
				break
			}
			nop = append(nop, 0x90)
		}

		code = append(code, nop...)
	}

	return code
}
