package main

import (
	"fmt"
	"hook"
	"golang.org/x/arch/x86/x86asm"
)

func main() {
	//code := []byte {0x64,0x48,0x8b,0x0c,0x25,0xf8,0xff,0xff,0xff,0x48,0x3b,0x61}
	//code := []byte {0x48,0x3b,0x61,0x10}
	code := []byte {0x8d,0x6c,0x24,0x10}
	//code := []byte {0x0f,0x86,0xa1,0x00, 0x00,0x00,0x00,0x00}
	//code := []byte {0xe8,0x8f,0x89,0xe0,0xff}
	inst, err := x86asm.Decode(code, 64)
	if err != nil {
		fmt.Printf("decode failed\n")
		return
	}

	fmt.Printf("op:%s,code:%x,len:%d,prefix:", inst.Op.String(), inst.Opcode, inst.Len)
	for _, v := range inst.Prefix {
		if v == 0 {
			break
		}
		fmt.Printf(" %s",v.String())
	}
	fmt.Printf(",args:")
	for _, v := range inst.Args {
		if v == nil {
			break
		}
		fmt.Printf(" %s",v.String())
	}

	fmt.Printf("\n")

	fullInstLen := hook.GetInsLenGreaterThan(code, 11)
	fmt.Printf("full inst len:%d\n", fullInstLen)
}

