package main

import (
	"bytes"
	"fmt"
	"gohook"
	"golang.org/x/arch/x86/x86asm"
)

//go:noinline
func foo1(v1 int, v2 string) int {
	fmt.Printf("foo1:%d(%s)\n", v1, v2)
	return v1 + 42
}

func foo2(v1 int, v2 string) int {
	fmt.Printf("foo2:%d(%s)\n", v1, v2)
	v1 = foo3(100, "not calling foo3")
	return v1 + 4200
}

func foo3(v1 int, v2 string) int {
	fmt.Printf("foo3:%d(%s)\n", v1, v2)
	return v1 + 10000
}

func TestAsm() {
	fmt.Printf("start testing...\n")

	ret1 := foo1(23, "sval for foo1")

	gohook.Hook(foo1, foo2, foo3)

	ret2 := foo1(23, "sval for foo1")

	fmt.Printf("r1:%d, r2:%d\n", ret1, ret2)
}

func myBuffLen(b *bytes.Buffer) int {
	return 233
}

func myBuffLenTramp(b *bytes.Buffer) int {
	fmt.Printf("start testing...\n")
	fmt.Printf("start testing...\n")
	fmt.Printf("start testing...\n")
	fmt.Printf("start testing...\n")
	fmt.Printf("start testing...\n")
	fmt.Printf("start testing...\n")
	return 1000
}

func main() {
	TestStackGrowth()
	buff := bytes.NewBufferString("abcd")
	fmt.Printf("len(buff):%d\n", buff.Len())

	err1 := gohook.HookMethod(buff, "Len", myBuffLen, myBuffLenTramp)
	if err1 != nil {
		fmt.Printf("errors:%s\n", err1.Error())
	}

	TestAsm()

	//code := []byte {0x64,0x48,0x8b,0x0c,0x25,0xf8,0xff,0xff,0xff,0x48,0x3b,0x61}
	//code := []byte {0x48,0x3b,0x61,0x10}
	//code := []byte {0x8d,0x6c,0x24,0x10}
	//code := []byte{0x64, 0x48, 0x8b, 0xc, 0x25, 0xf8, 0xff, 0xff, 0xff}
	code := []byte{0xC7, 0x44, 0x24, 0xFC, 0x01, 0x02, 0x03, 0x04}
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
		fmt.Printf(" %s", v.String())
	}
	fmt.Printf(",args:")
	for _, v := range inst.Args {
		if v == nil {
			break
		}
		fmt.Printf(" %s", v.String())
	}

	fmt.Printf("\n")

	fullInstLen := gohook.GetInsLenGreaterThan(gohook.GetArchMode(), code, 11)
	fmt.Printf("full inst len:%d\n", fullInstLen)
}

func victim(a, b, c int, e, f, g string) int {
	if a > 100 {
		return 42
	}

	var someBigStackArray [4096]byte // to occupy stack, don't let it escape
	for i := 0; i < len(someBigStackArray); i++ {
		someBigStackArray[i] = byte((a ^ b) & (i ^ c))
	}

	if (a % 2) != 0 {
		someBigStackArray[200] = 0xe9
	}

	fmt.Printf("calling real victim() (%s,%s,%s,%x):%dth\n", e, f, g, someBigStackArray[200], a)

	return 1 + victim(a+1, b-1, c-1, e, f, g)
}

func victimTrampoline(a, b, c int, e, f, g string) int {
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", e, f, g, a, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", e, f, g, a, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", e, f, g, a, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", e, f, g, a, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", e, f, g, a, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", e, f, g, a, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", e, f, g, a, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", e, f, g, a, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", e, f, g, a, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", e, f, g, a, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", e, f, g, a, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", e, f, g, a, 0x23)

	for {
		if (a % 2) != 0 {
			fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", a, e, f, g, 0x23)
		} else {
			a++
		}

		if a+b > 100 {
			break
		}

		buff := bytes.NewBufferString("something weird")
		fmt.Printf("len:%d\n", buff.Len())
	}

	return 1
}

func victimReplace(a, b, c int, e, f, g string) int {
	fmt.Printf("victimReplace sends its regard\n")
	ret := 0
	if a > 100 {
		ret = 100000
	}

	return ret + victimTrampoline(a, b, c, e, f, g)
}

func TestStackGrowth() {
	gohook.SetMinJmpCodeSize(64)
	defer gohook.SetMinJmpCodeSize(0)

	gohook.Hook(victim, victimReplace, victimTrampoline)

	victim(0, 1000, 100000, "ab", "miliao", "see")
}
