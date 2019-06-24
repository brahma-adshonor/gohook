package gohook

import (
	"fmt"
	"reflect"
	"unsafe"
)

func dummy(v int) string {
	return fmt.Sprintf("some text:%d", v)
}

type CodeInfo struct {
	How            string
	Origin         []byte
	Fix            []CodeFix
	TrampolineOrig []byte
}

func makeSliceFromPointer(p uintptr, length int) []byte {
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: p,
		Len:  length,
		Cap:  length,
	}))
}

func GetFuncInsSize(f interface{}) uint32 {
	sz := uint32(0)
	ptr := reflect.ValueOf(f).Pointer()
	if elfInfo != nil {
		sz, _ = elfInfo.GetFuncSize(ptr)
	}

	if sz == 0 {
		sz, _ = GetFuncSizeByGuess(GetArchMode(), ptr, true)
	}

	return sz
}

func CopyFunction(from, to interface{}, info *CodeInfo) ([]byte, error) {
	s := reflect.ValueOf(from).Pointer()
	d := reflect.ValueOf(to).Pointer()

	mode := GetArchMode()
	sz1 := getFuncSize(mode, s, true)
	sz2 := getFuncSize(mode, d, true)
	return doCopyFunction(mode, s, d, sz1, sz2, info)
}

func getFuncSize(mode int, addr uintptr, minimal bool) uint32 {
	sz := uint32(0)
	if elfInfo != nil {
		sz, _ = elfInfo.GetFuncSize(addr)
	}

	var err error
	if sz == 0 {
		sz, err = GetFuncSizeByGuess(mode, addr, minimal)
		if err != nil {
			return 0
		}
	}

	return sz
}

func doFixFuncInplace(mode int, addr, to uintptr, funcSz, to_sz int, info *CodeInfo) error {
	fix, err := fixFuncInstructionInplace(mode, addr, to, funcSz, to_sz)
	if err != nil {
		return err
	}

	total_len := 0
	for _, f := range fix {
		total_len += len(f.Code)
	}

	origin := makeSliceFromPointer(addr, int(total_len))
	sf := make([]byte, total_len)
	copy(sf, origin)

	for _, f := range fix {
		CopyInstruction(f.Addr, f.Code)
	}

	info.Fix = fix
	info.Origin = sf
	return nil
}

func doCopyFunction(mode int, from, to uintptr, sz1, sz2 uint32, info *CodeInfo) ([]byte, error) {
	if sz1 > sz2+1 { // add trailing int3 to the end
		return nil, fmt.Errorf("source addr:%x, target addr:%x, sizeof source func(%d) > sizeof of target func(%d)", from, to, sz1, sz2)
	}

	fix, err2 := copyFuncInstruction(mode, from, to, int(sz1))
	if err2 != nil {
		return nil, err2
	}

	origin := makeSliceFromPointer(to, int(sz2))
	sf := make([]byte, sz2)
	copy(sf, origin)

	curAddr := to
	for _, f := range fix {
		CopyInstruction(curAddr, f.Code)
		f.Addr = curAddr
		curAddr += uintptr(len(f.Code))
	}

	info.Fix = fix
	return sf, nil
}

func hookFunction(mode int, target, replace, trampoline uintptr) (*CodeInfo, error) {
	info := &CodeInfo{}
	jumpcode := genJumpCode(mode, replace, target)

	insLen := len(jumpcode)
	if trampoline != uintptr(0) {
		f := makeSliceFromPointer(target, len(jumpcode)*2)
		insLen = GetInsLenGreaterThan(mode, f, len(jumpcode))
	}

	// target slice
	ts := makeSliceFromPointer(target, insLen)
	info.Origin = make([]byte, len(ts))
	copy(info.Origin, ts)

	info.How = "jump"

	if trampoline != uintptr(0) {
		sz := uint32(0)
		if elfInfo != nil {
			sz, _ = elfInfo.GetFuncSize(target)
		}

		fix_trampoline := true
		fix, err := FixTargetFuncCode(mode, target, sz, trampoline, insLen)

		if err != nil {
			sz1 := getFuncSize(mode, target, false)
			sz2 := getFuncSize(mode, trampoline, false)
			if sz1 <= 0 || sz2 <= 0 {
				return nil, fmt.Errorf("failed calc func size")
			}

			err1 := doFixFuncInplace(mode, target, trampoline, int(sz1), insLen, info)
			if err1 != nil {
				info.How = "copy"
				origin, err2 := doCopyFunction(mode, target, trampoline, sz1, sz2, info)
				if err2 != nil {
					return nil, fmt.Errorf("both fix/fix2/copy failed, fix:%s, fix2:%s, copy:%s", err.Error(), err1.Error(), err2.Error())
				}
				fix_trampoline = false
				info.TrampolineOrig = origin
			} else {
				insLen = GetInsLenGreaterThan(mode, info.Origin, len(jumpcode))
				ts = makeSliceFromPointer(target, insLen)
			}
		} else {
			info.How = "fix"
			for _, v := range fix {
				origin := makeSliceFromPointer(v.Addr, len(v.Code))
				f := make([]byte, len(v.Code))
				copy(f, origin)
				CopyInstruction(v.Addr, v.Code)
				v.Code = f
				info.Fix = append(info.Fix, v)
			}
		}

		if (fix_trampoline) {
			jumpcode2 := genJumpCode(mode, target+uintptr(insLen), trampoline+uintptr(insLen))
			f2 := makeSliceFromPointer(trampoline, insLen+len(jumpcode2)*2)
			insLen2 := GetInsLenGreaterThan(mode, f2, insLen+len(jumpcode2))
			info.TrampolineOrig = make([]byte, insLen2)
			ts2 := makeSliceFromPointer(trampoline, insLen2)
			copy(info.TrampolineOrig, ts2)
			CopyInstruction(trampoline, ts)
			CopyInstruction(trampoline+uintptr(insLen), jumpcode2)
		}
	}

	CopyInstruction(target, jumpcode)
	return info, nil
}

func printInstructionFix(v CodeFix, origin []byte) {
	fmt.Printf("addr:0x%x, code:", v.Addr)
	for _, c := range v.Code {
		fmt.Printf(" %x", c)
	}

	fmt.Printf(", origin:")
	for _, c := range origin {
		fmt.Printf(" %x", c)
	}
	fmt.Printf("\n")
}

func GetFuncAddr(f interface{}) uintptr {
	fv := reflect.ValueOf(f)
	return fv.Pointer()
}
