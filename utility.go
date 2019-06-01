package hook

import (
	"reflect"
	"syscall"
	"unsafe"
)

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

		if len(jumpcode) > 5 || sz > 0 {
			//if size of jumpcode == 5, there is no chance we will mess up with jmp instruction
			//in this case we better dont fix code if we can not get function size
			fix, err := FixTargetFuncCode(mode, target, sz, trampoline, insLen)
			if err != nil {
				return nil, err
			}

			for _, v := range fix {
				origin := makeSliceFromPointer(v.Addr, len(v.Code))
				f := make([]byte, len(v.Code))
				copy(f, origin)

				/*
				   // test code
				   fmt.Printf("addr:0x%x, code:", v.Addr)
				   for _, c := range v.Code {
				       fmt.Printf(" %x", c)
				   }

				   fmt.Printf(", origin:")
				   for _, c := range f {
				       fmt.Printf(" %x", c)
				   }
				   fmt.Printf("\n")
				   // end test code
				*/

				CopyInstruction(v.Addr, v.Code)
				v.Code = f
				info.Fix = append(info.Fix, v)
			}
		}

		CopyInstruction(trampoline, ts)
	}

	CopyInstruction(target, jumpcode)

	if trampoline != uintptr(0) {
		jumpcode = genJumpCode(mode, target+uintptr(insLen), trampoline+uintptr(insLen))
		insLen2 := len(jumpcode)
		info.TrampolineOrig = make([]byte, len(info.Origin)+insLen2)

		ts2 := makeSliceFromPointer(trampoline, len(info.Origin)+insLen2)
		copy(info.TrampolineOrig, ts2)
		CopyInstruction(trampoline+uintptr(insLen), jumpcode)
	}

	return info, nil
}
