package hook

import (
	"fmt"
	"reflect"
	"syscall"
	"unsafe"
)

func dummy(v int) string {
	return fmt.Sprintf("some text:%d", v)
}

type CodeInfo struct {
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

func CopyInstruction(location uintptr, data []byte) {
	f := makeSliceFromPointer(location, len(data))
	setPageWritable(location, len(data), syscall.PROT_READ|syscall.PROT_WRITE|syscall.PROT_EXEC)
	copy(f, data[:])
	setPageWritable(location, len(data), syscall.PROT_READ|syscall.PROT_EXEC)
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

	if trampoline != uintptr(0) {
		sz := uint32(0)
		if elfInfo != nil {
			sz, _ = elfInfo.GetFuncSize(target)
		}

		fix, err := FixTargetFuncCode(mode, target, sz, trampoline, insLen)
		if err != nil {
			return nil, err
		}

		for _, v := range fix {
			origin := makeSliceFromPointer(v.Addr, len(v.Code))
			f := make([]byte, len(v.Code))
			copy(f, origin)

			// printInstructionFix(v, f)

			CopyInstruction(v.Addr, v.Code)
			v.Code = f
			info.Fix = append(info.Fix, v)
		}

		jumpcode2 := genJumpCode(mode, target+uintptr(insLen), trampoline+uintptr(insLen))
		f := makeSliceFromPointer(trampoline, insLen+len(jumpcode2)*2)
		insLen2 := GetInsLenGreaterThan(mode, f, insLen+len(jumpcode2))
		info.TrampolineOrig = make([]byte, insLen2)
		ts2 := makeSliceFromPointer(trampoline, insLen2)
		copy(info.TrampolineOrig, ts2)
		CopyInstruction(trampoline, ts)
		CopyInstruction(trampoline+uintptr(insLen), jumpcode2)
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
