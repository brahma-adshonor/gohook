package gohook

import (
	"bytes"
	"fmt"
	"reflect"
	"runtime"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func myPrintf(f string, a ...interface{}) (n int, err error) {
	myPrintfTramp("prefixed by miliao -- ")
	return myPrintfTramp(f, a...)
}

//go:noinline
func myPrintfTramp(f string, a ...interface{}) (n int, err error) {
	fmt.Printf("hello")
	fmt.Printf("hello")
	fmt.Printf("hello")
	fmt.Printf("hello")
	fmt.Printf("hello")
	return fmt.Printf("hello")
}

func init() {
	fmt.Printf("test file init()\n")
	err := Hook(fmt.Printf, myPrintf, myPrintfTramp)
	if err != nil {
		fmt.Printf("err:%s\n", err.Error())
	} else {
		fmt.Printf("hook fmt.Printf() done\n")
	}

	fmt.Printf("debug info for init():%s\n", ShowDebugInfo())
}

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

//go:noinline
func foo3(v1 int, v2 string) int {
	fmt.Printf("foo3:%d(%s)\n", v1, v2)
	return v1 + 10000
}

func myByteContain(a, b []byte) bool {
	fmt.Printf("calling fake bytes.Contain()\n")
	return false
}

func TestHook(t *testing.T) {
	ResetFuncPrologue()

	fmt.Printf("start testing...\n")

	ret1 := foo1(23, "sval for foo1")
	assert.Equal(t, 65, ret1)

	err := Hook(foo1, foo2, foo3)
	assert.Nil(t, err)

	ret2 := foo1(23, "sval for foo1")
	assert.Equal(t, 4342, ret2)

	ret4 := foo3(100, "vvv")
	assert.Equal(t, 142, ret4)

	UnHook(foo1)
	ret3 := foo1(23, "sval for foo1")
	assert.Equal(t, 65, ret3)

	ret5 := foo3(100, "vvv")
	assert.Equal(t, 10100, ret5)

	ret6 := bytes.Contains([]byte{1, 2, 3}, []byte{2, 3})
	assert.Equal(t, true, ret6)
	err = Hook(bytes.Contains, myByteContain, nil)
	assert.Nil(t, err)

	fun := bytes.Contains // prevent inline
	ret7 := fun([]byte{1, 2, 3}, []byte{2, 3})

	assert.Equal(t, false, ret7)
	UnHook(bytes.Contains)
	ret8 := bytes.Contains([]byte{1, 2, 3}, []byte{2, 3})
	assert.Equal(t, true, ret8)
}

func myBuffLen(b *bytes.Buffer) int {
	fmt.Println("calling myBuffLen")
	return 0 + myBuffLenTramp(b)
}

//go:noinline
func myBuffLenTramp(b *bytes.Buffer) int {
	fmt.Println("calling myBuffLenTramp")
	return 1000
}

func myBuffGrow(b *bytes.Buffer, n int) {
	fmt.Println("fake buffer grow func")
}

func myBuffWriteString(b *bytes.Buffer, s string) (int, error) {
	fmt.Printf("fake buffer WriteString func, s:%s\n", s)

	l, _ := myBuffWriteStringTramp(b, s)
	return 1000 + l, nil
}

func myBuffWriteStringTramp(b *bytes.Buffer, s string) (int, error) {
	fmt.Printf("fake buffer WriteString tramp, s:%s\n", s)
	fmt.Printf("fake buffer WriteString tramp, s:%s\n", s)
	fmt.Printf("fake buffer WriteString tramp, s:%s\n", s)
	fmt.Printf("fake buffer WriteString tramp, s:%s\n", s)
	fmt.Printf("fake buffer WriteString tramp, s:%s\n", s)
	fmt.Printf("fake buffer WriteString tramp, s:%s\n", s)
	return 0, nil
}

func TestInstanceHook(t *testing.T) {
	ResetFuncPrologue()
	buff1 := bytes.NewBufferString("abcd")
	assert.Equal(t, 4, buff1.Len())

	err1 := HookMethod(buff1, "Grow", myBuffGrow, nil)
	err2 := HookMethod(buff1, "Len", myBuffLen, myBuffLenTramp)

	assert.Nil(t, err1)
	assert.Nil(t, err2)

	assert.Equal(t, 4, buff1.Len()) // Len() is inlined
	buff1.Grow(233)                 // no grow
	assert.Equal(t, 4, buff1.Len()) // Len() is inlined

	err3 := HookMethod(buff1, "WriteString", myBuffWriteString, myBuffWriteStringTramp)
	assert.Nil(t, err3)

	sz1, _ := buff1.WriteString("miliao")
	assert.Equal(t, 1006, sz1)
	assert.Equal(t, 10, buff1.Len()) // Len() is inlined

	err4 := UnHookMethod(buff1, "WriteString")
	assert.Nil(t, err4)

	flen := buff1.Len

	sz2, _ := buff1.WriteString("miliao")
	assert.Equal(t, 6, sz2)
	assert.Equal(t, 16, flen()) // Len() is inlined

	sz3, _ := myBuffWriteStringTramp(nil, "sssssss")
	assert.Equal(t, 0, sz3)
}

func TestGetInsLenGreaterThan(t *testing.T) {
	ResetFuncPrologue()
	c1 := []byte{0x64, 0x48, 0x8b, 0x0c, 0x25, 0xf8}
	c2 := []byte{0x64, 0x48, 0x8b, 0x0c, 0x25, 0xf8, 0xff, 0xff, 0xff}

	r1 := GetInsLenGreaterThan(64, c1, len(c1)-2)
	assert.Equal(t, 0, r1)
	r2 := GetInsLenGreaterThan(64, c2, len(c2)-2)
	assert.Equal(t, len(c2), r2)
	r22 := GetInsLenGreaterThan(64, c2, len(c2))
	assert.Equal(t, len(c2), r22)
	r23 := GetInsLenGreaterThan(64, c2, len(c2)+2)
	assert.Equal(t, 0, r23)

	c3 := []byte{0x64, 0x48, 0x8b, 0x0c, 0x25, 0xf8, 0xff, 0xff, 0xff, 0x48, 0x3b, 0x41, 0x10}
	r3 := GetInsLenGreaterThan(64, c3, len(c2)+2)
	assert.Equal(t, len(c3), r3)
	r32 := GetInsLenGreaterThan(64, c3, len(c2)-2)
	assert.Equal(t, len(c2), r32)
}

func TestFixOneInstructionForTwoByteJmp(t *testing.T) {
	ResetFuncPrologue()
	// jump from within patching erea to outside, negative fix
	c1 := []byte{0x75, 0x40} // jne 64

	l1, t1, r1 := FixOneInstruction(64, false, 10, 12, c1, 100, 8)

	assert.Equal(t, 2, l1)
	assert.Equal(t, FT_CondJmp, t1)
	assert.Equal(t, c1[0], r1[0])
	assert.Equal(t, int8(-26), int8(r1[1]))

	// jump from within patching erea to outside, positive fix
	l2, t2, r2 := FixOneInstruction(64, false, 10, 12, c1, 26, 8)

	assert.Equal(t, 2, l2)
	assert.Equal(t, FT_CondJmp, t2)
	assert.Equal(t, c1[0], r2[0])
	assert.Equal(t, int8(48), int8(r2[1]))

	//overflow test
	l3, t3, r3 := FixOneInstruction(64, false, 10, 12, c1, 1000, 8)

	assert.Equal(t, 2, l3)
	assert.Equal(t, FT_OVERFLOW, t3)
	assert.Equal(t, c1[0], r3[0])
	assert.Equal(t, c1[1], r3[1])

	// overflow test2
	c32 := []byte{0x75, 0x7e} // jne 0x7e
	l32, t32, r32 := FixOneInstruction(64, false, 30, 32, c32, 10, 8)

	assert.Equal(t, 2, l32)
	assert.Equal(t, FT_OVERFLOW, t32)
	assert.Equal(t, c32[0], r32[0])
	assert.Equal(t, c32[1], r32[1])

	// jump from outside patching erea to outside of patching erea
	l4, t4, r4 := FixOneInstruction(64, false, 10, 18, c1, 100, 4)

	assert.Equal(t, 2, l4)
	assert.Equal(t, FT_SKIP, t4)
	assert.Equal(t, c1[0], r4[0])
	assert.Equal(t, c1[1], r4[1])

	// jump from outside patching erea to within patching erea
	c2 := []byte{0x75, 0xe6} // jne -26
	l5, t5, r5 := FixOneInstruction(64, false, 10, 38, c2, 100, 8)

	assert.Equal(t, 2, l5)
	assert.Equal(t, FT_CondJmp, t5)
	assert.Equal(t, c2[0], r5[0])
	assert.Equal(t, 64, int(r5[1]))

	// jump within patching erea
	c3 := []byte{0x75, 0x06} // jne 6
	l6, t6, r6 := FixOneInstruction(64, false, 10, 12, c3, 100, 11)

	assert.Equal(t, 2, l6)
	assert.Equal(t, FT_SKIP, t6)
	assert.Equal(t, c3[0], r6[0])
	assert.Equal(t, c3[1], r6[1])

	// sign test, from outside to outside
	c4 := []byte{0x7c, 0xcd} // jne -51
	l7, t7, r7 := FixOneInstruction(64, false, 10, 83, c4, 1000, 10)

	assert.Equal(t, 2, l7)
	assert.Equal(t, FT_SKIP, t7)
	assert.Equal(t, c4[0], r7[0])
	assert.Equal(t, c4[1], r7[1])
}

func byteToInt32(d []byte) int32 {
	v := int32(uint32(d[0]) | (uint32(d[1]) << 8) | (uint32(d[2]) << 16) | (uint32(d[3]) << 24))
	return v
}

func TestFixOneInstructionForSixByteJmp(t *testing.T) {
	ResetFuncPrologue()
	// jump from within patching erea to outside, negative fix
	c1 := []byte{0x0f, 0x8d, 0x10, 0x00, 0x00, 0x00} // jge 16

	l1, t1, r1 := FixOneInstruction(64, false, 20, 22, c1, 100, 8)
	assert.Equal(t, 6, l1)
	assert.Equal(t, FT_CondJmp, t1)
	assert.Equal(t, c1[0], r1[0])
	assert.Equal(t, c1[1], r1[1])

	assert.Equal(t, int32(-64), byteToInt32(r1[2:]))

	// jump from within patching erea to outside, positive fix
	c2 := []byte{0x0f, 0x8d, 0x40, 0x00, 0x00, 0x00} // jge 64

	l2, t2, r2 := FixOneInstruction(64, false, 2, 4, c2, 32, 9)
	assert.Equal(t, 6, l2)
	assert.Equal(t, FT_CondJmp, t2)
	assert.Equal(t, c2[0], r2[0])
	assert.Equal(t, c2[1], r2[1])

	assert.Equal(t, int32(34), byteToInt32(r2[2:]))

	// overflow test
	c3 := []byte{0x0f, 0x8d, 0xfe, 0xff, 0xff, 0x7f} // jge 64

	l3, t3, r3 := FixOneInstruction(64, false, 10000, 10004, c3, 100, 16)
	assert.Equal(t, 6, l3)
	assert.Equal(t, FT_OVERFLOW, t3)
	assert.Equal(t, c3[0], r3[0])
	assert.Equal(t, c3[1], r3[1])
	assert.Equal(t, c3[2], r3[2])
	assert.Equal(t, c3[3], r3[3])
	assert.Equal(t, c3[4], r3[4])
	assert.Equal(t, c3[5], r3[5])

	// jump from outside patching erea to outside of patching erea
	c4 := []byte{0x0f, 0x8d, 0x40, 0x00, 0x00, 0x00} // jge 64

	l4, t4, r4 := FixOneInstruction(64, false, 10, 33, c4, 22, 9)
	assert.Equal(t, 6, l4)
	assert.Equal(t, FT_SKIP, t4)
	assert.Equal(t, c4[0], r4[0])
	assert.Equal(t, c4[1], r4[1])
	assert.Equal(t, c4[2], r4[2])
	assert.Equal(t, c4[3], r4[3])
	assert.Equal(t, c4[4], r4[4])
	assert.Equal(t, c4[5], r4[5])

	// jump from outside patching erea to within patching erea
	c5 := []byte{0x0f, 0x85, 0xce, 0xff, 0xff, 0xff} // jne -50

	l5, t5, r5 := FixOneInstruction(64, false, 10, 60, c5, 1000, 9)
	assert.Equal(t, 6, l5)
	assert.Equal(t, FT_CondJmp, t5)
	assert.Equal(t, c5[0], r5[0])
	assert.Equal(t, c5[1], r5[1])

	assert.Equal(t, int32(940), byteToInt32(r5[2:]))

	// jump within patching erea
	c6 := []byte{0x0f, 0x85, 0x10, 0x00, 0x00, 0x00} // jne 16

	l6, t6, r6 := FixOneInstruction(64, false, 10, 12, c6, 1000, 30)
	assert.Equal(t, 6, l6)
	assert.Equal(t, FT_SKIP, t6)
	assert.Equal(t, c6[0], r6[0])
	assert.Equal(t, c6[1], r6[1])
	assert.Equal(t, c6[2], r6[2])
	assert.Equal(t, c6[3], r6[3])
	assert.Equal(t, c6[4], r6[4])
	assert.Equal(t, c6[5], r6[5])
}

func TestFixOneInstructionForFixByteJmp(t *testing.T) {
	// jump from within patching erea to outside, negative fix
	c1 := []byte{0xe9, 0x10, 0x00, 0x00, 0x00} // jmp 16

	l1, t1, r1 := FixOneInstruction(64, false, 20, 22, c1, 100, 8)
	assert.Equal(t, 5, l1)
	assert.Equal(t, FT_JMP, t1)
	assert.Equal(t, c1[0], r1[0])
	assert.Equal(t, int32(-64), byteToInt32(r1[1:]))

	// jump from within patching erea to outside, positive fix
	c2 := []byte{0xe9, 0x40, 0x00, 0x00, 0x00} // jmp 64

	l2, t2, r2 := FixOneInstruction(64, false, 2, 4, c2, 32, 9)
	assert.Equal(t, 5, l2)
	assert.Equal(t, FT_JMP, t2)
	assert.Equal(t, c2[0], r2[0])
	assert.Equal(t, int32(34), byteToInt32(r2[1:]))

	// overflow test
	c3 := []byte{0xe9, 0xfe, 0xff, 0xff, 0x7f} // jmp 64

	l3, t3, r3 := FixOneInstruction(64, false, 10000, 10004, c3, 100, 16)
	assert.Equal(t, 5, l3)
	assert.Equal(t, FT_OVERFLOW, t3)
	assert.Equal(t, c3[0], r3[0])
	assert.Equal(t, c3[1], r3[1])
	assert.Equal(t, c3[2], r3[2])
	assert.Equal(t, c3[3], r3[3])
	assert.Equal(t, c3[4], r3[4])

	// jump from outside patching erea to outside of patching erea
	c4 := []byte{0xe9, 0x40, 0x00, 0x00, 0x00} // jmp 64

	l4, t4, r4 := FixOneInstruction(64, false, 10, 33, c4, 22, 9)
	assert.Equal(t, 5, l4)
	assert.Equal(t, FT_SKIP, t4)
	assert.Equal(t, c4[0], r4[0])
	assert.Equal(t, c4[1], r4[1])
	assert.Equal(t, c4[2], r4[2])
	assert.Equal(t, c4[3], r4[3])
	assert.Equal(t, c4[4], r4[4])

	// jump from outside patching erea to within patching erea
	c5 := []byte{0xe9, 0xce, 0xff, 0xff, 0xff} // jmp -50

	l5, t5, r5 := FixOneInstruction(64, false, 10, 60, c5, 1000, 9)
	assert.Equal(t, 5, l5)
	assert.Equal(t, FT_JMP, t5)
	assert.Equal(t, c5[0], r5[0])
	assert.Equal(t, int32(940), byteToInt32(r5[1:]))

	// jump within patching erea
	c6 := []byte{0xe9, 0x10, 0x00, 0x00, 0x00} // jmp 16

	l6, t6, r6 := FixOneInstruction(64, false, 10, 12, c6, 1000, 30)
	assert.Equal(t, 5, l6)
	assert.Equal(t, FT_SKIP, t6)
	assert.Equal(t, c6[0], r6[0])
	assert.Equal(t, c6[1], r6[1])
	assert.Equal(t, c6[2], r6[2])
	assert.Equal(t, c6[3], r6[3])
	assert.Equal(t, c6[4], r6[4])

	// jump from outside to outside, sign test
	c7 := []byte{0xe8, 0xdc, 0xfb, 0xff, 0xff} // jmp -1060
	l7, t7, r7 := FixOneInstruction(64, false, 2000, 4100, c7, 10000, 30)
	assert.Equal(t, 5, l7)
	assert.Equal(t, FT_SKIP, t7)
	assert.Equal(t, c7[0], r7[0])
	assert.Equal(t, c7[1], r7[1])
	assert.Equal(t, c7[2], r7[2])
	assert.Equal(t, c7[3], r7[3])
	assert.Equal(t, c7[4], r7[4])
}

func TestFixFuncCode(t *testing.T) {
	p := []byte{0x64, 0x48, 0x8b, 0x0c, 0x25, 0xf8, 0xff, 0xff, 0xff} // move %fs:0xfffffffffffffff8, %rcx
	c1 := []byte{
		/*0:*/ 0x64, 0x48, 0x8b, 0x0c, 0x25, 0xf8, 0xff, 0xff, 0xff, // move %fs:0xfffffffffffffff8, %rcx   sz:9
		/*9:*/ 0x48, 0x8d, 0x44, 0x24, 0xe0, // lea    -0x20(%rsp),%rax             sz:5
		/*14:*/ 0x48, 0x3b, 0x41, 0x10, // cmp    0x10(%rcx),%rax              sz:4
		/*18:*/ 0x0f, 0x86, 0xc3, 0x01, 0x00, 0x00, // jbe    451                           sz:6
		/*24:*/ 0x48, 0x81, 0xec, 0xa0, 0x00, 0x00, 0x00, // sub    $0xa0,%rsp                   sz:7
		/*31:*/ 0x48, 0x8b, 0x9c, 0x24, 0xa8, 0x00, 0x00, 0x00, // mov    0xa8(%rsp),%rbx              sz:8
		/*39:*/ 0xe3, 0x02, // jmp 02                       sz:2
		/*41:*/ 0x90, // nop sz:1
		/*42:*/ 0x90, // nop sz:1
		/*43:*/ 0x90, // nop sz:1
		/*44:*/ 0x90, // nop sz:1
		//////////patching erea end: 45 bytes/////////////////////////////////////////
		/*45:*/ 0x48, 0x89, 0x5c, 0x24, 0x40, // mov    %rbx,0x40(%rsp)              sz:5
		/*50:*/ 0xe9, 0xd2, 0xff, 0xff, 0xff, // jmp -46      sz:5
		/*55:*/ 0x90, // nop                                  sz:1
		/*56:*/ 0x90, // nop                                  sz:1
		/*57:*/ 0x90, // nop                                  sz:1
		/*58:*/ 0x90, // nop                                  sz:1
	}

	SetFuncPrologue(64, []byte{0x64, 0x48, 0x8b, 0x0c, 0x25, 0xf8, 0xff, 0xff, 0xff, 0x48})
	sh1 := (*reflect.SliceHeader)((unsafe.Pointer(&c1)))

	move_sz := 45
	startAddr := sh1.Data
	toAddr := startAddr + 100000

	fix1, err1 := FixTargetFuncCode(64, startAddr, uint32(len(c1)), toAddr, move_sz)

	assert.Nil(t, err1)
	assert.Equal(t, 2, len(fix1))

	assert.Equal(t, startAddr+uintptr(18), fix1[0].Addr)
	assert.Equal(t, startAddr+uintptr(50), fix1[1].Addr)

	assert.Equal(t, 6, len(fix1[0].Code))
	assert.Equal(t, byte(0x0f), fix1[0].Code[0])
	assert.Equal(t, byte(0x86), fix1[0].Code[1])
	assert.Equal(t, int32(startAddr+451-toAddr), byteToInt32(fix1[0].Code[2:]))

	assert.Equal(t, 5, len(fix1[1].Code))
	assert.Equal(t, byte(0xe9), fix1[1].Code[0])
	assert.Equal(t, int32(toAddr+9-startAddr-50-5), byteToInt32(fix1[1].Code[1:]))

	c2 := append(c1, p...)
	sh2 := (*reflect.SliceHeader)((unsafe.Pointer(&c2)))
	startAddr = sh2.Data
	toAddr = startAddr + 100000

	fix2, err2 := FixTargetFuncCode(64, startAddr, 0, toAddr, move_sz)

	assert.Nil(t, err2)
	assert.Equal(t, 2, len(fix2))

	assert.Equal(t, startAddr+uintptr(18), fix2[0].Addr)
	assert.Equal(t, startAddr+uintptr(50), fix2[1].Addr)

	assert.Equal(t, 6, len(fix2[0].Code))
	assert.Equal(t, byte(0x0f), fix2[0].Code[0])
	assert.Equal(t, byte(0x86), fix2[0].Code[1])
	assert.Equal(t, int32(startAddr+451-toAddr), byteToInt32(fix2[0].Code[2:]))

	assert.Equal(t, 5, len(fix2[1].Code))
	assert.Equal(t, byte(0xe9), fix2[1].Code[0])
	assert.Equal(t, int32(toAddr+9-startAddr-50-5), byteToInt32(fix2[1].Code[1:]))
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

	return victim(a+1, b-1, c-1, e, f, g)
}

func victimTrampoline(a, b, c int, e, f, g string) int {
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)

	for {
		if (a % 2) != 0 {
			fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
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

func TestStackGrowth(t *testing.T) {
	SetMinJmpCodeSize(64)
	defer SetMinJmpCodeSize(0)

	ResetFuncPrologue()

	err := Hook(victim, victimReplace, victimTrampoline)
	assert.Nil(t, err)

	ret := victim(0, 1000, 100000, "ab", "miliao", "see")

	assert.Equal(t, 100042, ret)

	UnHook(victim)

	fmt.Printf("after unHook\n")
	victimReplace(98, 2, 3, "ab", "ef", "g")
}

func TestFuncSize(t *testing.T) {
	ResetFuncPrologue()

	addr1 := GetFuncAddr(victim)
	addr2 := GetFuncAddr(victimReplace)
	addr3 := GetFuncAddr(victimTrampoline)

	elf, err := NewElfInfo()
	hasElf := (err == nil)

	sz11, err11 := GetFuncSizeByGuess(GetArchMode(), addr1, true)
	assert.Nil(t, err11)

	if hasElf {
		sz1, err1 := elf.GetFuncSize(addr1)
		assert.Nil(t, err1)
		assert.Equal(t, sz1, sz11)
	} else {
		assert.True(t, sz11 > 0)
	}

	sz21, err21 := GetFuncSizeByGuess(GetArchMode(), addr2, true)
	assert.Nil(t, err21)

	if hasElf {
		sz2, err2 := elf.GetFuncSize(addr2)
		assert.Nil(t, err2)
		assert.Equal(t, sz2, sz21)
	}

	sz31, err31 := GetFuncSizeByGuess(GetArchMode(), addr3, true)
	assert.Nil(t, err31)

	if hasElf {
		sz3, err3 := elf.GetFuncSize(addr3)
		assert.Nil(t, err3)

		assert.Equal(t, sz3, sz31)
	}
}

func mySprintf(format string, a ...interface{}) string {
	addr1 := GetFuncAddr(victim)
	addr2 := GetFuncAddr(victimReplace)
	addr3 := GetFuncAddr(victimTrampoline)

	elf, err := NewElfInfo()
	fmt.Println("show:", elf, err)

	sz1, err1 := elf.GetFuncSize(addr1)
	fmt.Println("show:", sz1, err1)

	sz11, err11 := GetFuncSizeByGuess(GetArchMode(), addr1, false)
	fmt.Println("show:", sz11, err11)

	sz2, err2 := elf.GetFuncSize(addr2)
	fmt.Println("show:", sz2, err2)
	sz21, err21 := GetFuncSizeByGuess(GetArchMode(), addr2, false)
	fmt.Println("show:", sz21, err21)

	sz3, err3 := elf.GetFuncSize(addr3)
	fmt.Println("show:", sz3, err3)
	sz31, err31 := GetFuncSizeByGuess(GetArchMode(), addr3, false)
	fmt.Println("show:", sz31, err31)

	return ""
}

func TestCopyFunc(t *testing.T) {
	ResetFuncPrologue()

	addr := GetFuncAddr(mySprintf)
	sz := GetFuncInstSize(mySprintf)

	tp := makeSliceFromPointer(addr, int(sz))
	txt := make([]byte, int(sz))
	copy(txt, tp)

	fs := "some random text, from %d,%S,%T"
	s1 := fmt.Sprintf(fs, 233, "miliao test sprintf", addr)

	info := &CodeInfo{}
	origin, err := CopyFunction(true, fmt.Sprintf, mySprintf, info)

	assert.Nil(t, err)
	assert.Equal(t, len(txt), len(origin))
	assert.Equal(t, txt, origin)

	s2 := mySprintf(fs, 233, "miliao test sprintf", addr)

	assert.Equal(t, s1, s2)

	addr2 := GetFuncAddr(fmt.Sprintf)
	sz2, _ := GetFuncSizeByGuess(GetArchMode(), addr2, true)
	sz3, _ := GetFuncSizeByGuess(GetArchMode(), addr, true)

	assert.Equal(t, sz2, sz3)
}

func inplaceFix(a, b, c int, e, f, g string) int {
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)

	for {
		if (a % 2) != 0 {
			fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
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

func TestFixInplace(t *testing.T) {
	d0 := int32(0x01b1)
	d1 := byte(0xb2) // byte(-78)
	d2 := byte(0xa8) // byte(-88)
	prefix1 := []byte{
		0x64, 0x48, 0x8b, 0x0c, 0x25, 0xf8, 0xff, 0xff, 0xff, // 9
		0x48, 0x3b, 0x61, 0x10, // 4
	}

	jc0 := []byte{
		0x0f, 0x86, byte(d0), byte(d0 >> 8), 0x00, 0x00, // jmp
	}

	prefix2 := []byte{
		0x48, 0x83, 0xec, 0x58, // 4
		0x48, 0x89, 0x6c, 0x24, 0x50, // 5
		0x48, 0x8d, 0x6c, 0x24, 0x50, // 5
		0x90,                                     // 1
		0x48, 0x8b, 0x05, 0xc7, 0x5f, 0x16, 0x00, // 7
		0x48, 0x8d, 0x0d, 0x50, 0x85, 0x07, 0x00, // 7
		0x48, 0x89, 0x0c, 0x24, // 4
		0x48, 0x89, 0x44, 0x24, 0x08, // 5
		0x48, 0x8d, 0x05, 0x9f, 0xe0, 0x04, 0x00, // 7
		0x48, 0x89, 0x44, 0x24, 0x10, // 5
		0x48, 0xc7, 0x44, 0x24, 0x18, 0x07, 0x00, 0x00, 0x00, // 9
	}
	// totoal 78 bytes

	// short jump
	jc1 := []byte{0xeb, d1} // 2

	mid := []byte{
		0x0f, 0x57, 0xc0, // 3
		0x0f, 0x11, 0x44, 0x24, 0x28, // 5
	}

	jc2 := []byte{
		// condition jump
		0x77, d2, // 2
	}

	posfix := []byte{
		// trailing
		0xcc, 0xcc, 0xcc, 0xcc,
		0xcc, 0xcc, 0xcc, 0xcc,
		0xcc, 0xcc, 0xcc, 0xcc,
		0xcc, 0xcc, 0xcc, 0xcc,
	}

	fc := append(append(append(append(append(append(prefix1, jc0...), prefix2...), jc1...), mid...), jc2...), posfix...)

	info := &CodeInfo{}
	addr := GetFuncAddr(inplaceFix)
	size := len(fc)
	mvSize := 0x09
	toAddr := GetFuncAddr(inplaceFix2)

	curAddr1 := addr + uintptr(78)
	curAddr2 := addr + uintptr(78) + uintptr(10)

	CopyInstruction(addr, fc)

	fs := makeSliceFromPointer(addr, len(fc))
	raw := make([]byte, len(fc))
	copy(raw, fs)

	fmt.Printf("src func:%x, target func:%x\n", addr, toAddr)

	err := doFixFuncInplace(64, addr, toAddr, int(size), mvSize, info, 5)

	assert.Nil(t, err)
	assert.True(t, len(raw) >= len(info.Origin))
	raw = raw[:len(info.Origin)]
	assert.Equal(t, raw, info.Origin)
	assert.Equal(t, 18, len(info.Fix))
	assert.Equal(t, prefix1[:5], fs[:5])

	off0 := d0 + 4
	fix0, _ := adjustInstructionOffset(jc0, int64(off0))
	fmt.Printf("inplace fix, off0:%x, sz:%d\n", off0, len(fix0))

	to1 := curAddr1 + uintptr(2) + uintptr(int32(int8(d1)))
	newTo1 := toAddr + to1 - addr
	off1 := int64(newTo1 - (curAddr1 - uintptr(4)) - 5)
	fix1, _ := translateJump(off1, jc1)
	fmt.Printf("inplace fix, off1:%x, sz:%d\n", off1, len(fix1))

	to2 := curAddr2 + uintptr(2) + uintptr(int32(int8(d2)))
	newTo2 := toAddr + to2 - addr
	off2 := int64(newTo2 - (curAddr2 + uintptr(3) - uintptr(4)) - 6)
	fix2, _ := translateJump(off2, jc2)
	fmt.Printf("inplace fix, off2:%x, sz:%d\n", off2, len(fix2))

	fc2 := append(append(append(append(append(append(append(prefix1[:5], prefix1[9:]...), fix0...), prefix2...), fix1...), mid...), fix2...), posfix...)
	assert.Equal(t, len(fc)-4+3+4, len(fc2))

	fs = makeSliceFromPointer(addr, len(fc2)-len(posfix))
	assert.Equal(t, fc2[:len(fc2)-len(posfix)], fs)
}

func inplaceFix2(a, b, c int, e, f, g string) int {
	fmt.Printf("calling inplacefix2()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling inplacefix2()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling inplacefix2()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling inplacefix2()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling inplacefix2()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling inplacefix2()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling inplacefix2()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling inplacefix2()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling inplacefix2()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling inplacefix2()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)

	for {
		if (a % 2) != 0 {
			fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
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

func foo_for_inplace_fix(id string) string {
	c := 0
	for {
		fmt.Printf("calling victim\n")
		if id == "miliao" {
			return "done"
		}

		c++
		if c > len(id) {
			break
		}
	}

	fmt.Printf("len:%d\n", len(id))
	return id + "xxx"
}

func foo_for_inplace_fix_delimiter(id string) string {
	for {
		fmt.Printf("calling victim trampoline")
		if id == "miliao" {
			return "done"
		}
		break
	}

	ret := "miliao"
	ret += foo_for_inplace_fix("test")
	ret += foo_for_inplace_fix("test")
	ret += foo_for_inplace_fix("test")
	ret += foo_for_inplace_fix("test")

	fmt.Printf("len1:%d\n", len(id))
	fmt.Printf("len2:%d\n", len(ret))

	return id + ret
}

func foo_for_inplace_fix_replace(id string) string {
	c := 0
	for {
		fmt.Printf("calling foo_for_inplace_fix_replace\n")
		if id == "miliao" {
			return "done"
		}
		c++
		if c > len(id) {
			break
		}
	}

	// TODO uncomment following
	foo_for_inplace_fix_trampoline("miliao")

	fmt.Printf("len:%d\n", len(id))
	return id + "xxx2"
}

func foo_for_inplace_fix_trampoline(id string) string {
	c := 0
	for {
		fmt.Printf("calling foo_for_inplace_fix_trampoline\n")
		if id == "miliao" {
			return "done"
		}
		c++
		if c > len(id) {
			break
		}
	}

	fmt.Printf("len:%d\n", len(id))
	return id + "xxx3"
}

func TestInplaceFixAtMoveArea(t *testing.T) {
	code := []byte{
		/*
			0x48, 0x8b, 0x48, 0x08, // mov 0x8(%rax),%rcx
			0x74, 0x4, // jbe
			0x48, 0x8b, 0x48, 0x18, // sub 0x18(%rax), %rcx
			0x48, 0x89, 0x4c, 0x24, 0x10, // %rcx, 0x10(%rsp)
			0xc3, // retq
			0xcc, 0xcc,
		*/
		0x90, 0x90,
		0xeb, 0x04, // jmp 4
		0x90, 0x90, 0x90, 0x90, 0x90,
		0x90, 0x90, 0x90, 0x90, 0x90,
		0xc3,
		0x74, 0xf0, // jbe -16
		0xcc, 0xcc, 0xcc, 0xcc,
		0xcc, 0xcc, 0xcc, 0xcc,
	}

	target := GetFuncAddr(foo_for_inplace_fix)
	replace := GetFuncAddr(foo_for_inplace_fix_replace)
	trampoline := GetFuncAddr(foo_for_inplace_fix_trampoline)

	assert.True(t, isByteOverflow((int32)(trampoline-target)))

	CopyInstruction(target, code)

	fmt.Printf("short call target:%x, replace:%x, trampoline:%x\n", target, replace, trampoline)
	err1 := Hook(foo_for_inplace_fix, foo_for_inplace_fix_replace, foo_for_inplace_fix_trampoline)
	assert.Nil(t, err1)

	fmt.Printf("debug info:%s\n", ShowDebugInfo())

	msg1 := foo_for_inplace_fix("txt")

	fmt.Printf("calling foo inplace fix func\n")

	assert.Equal(t, "txtxxx2", msg1)

	sz1 := 5
	na1 := trampoline + uintptr(2)
	ta1 := target + uintptr(2+5+4-3)
	off1 := ta1 - (na1 + uintptr(sz1))

	sz2 := 6
	na2 := target + uintptr(15+3-3)
	ta2 := trampoline + uintptr(1)
	off2 := ta2 - (na2 + uintptr(sz2))

	fmt.Printf("off1:%x, off2:%x\n", off1, off2)

	ret := []byte{
		0x90, 0x90,
		0xe9, 0x74, 0xfc, 0xff, 0xff,
		0x90, 0x90, 0x90, 0x90, 0x90,
		0x90, 0x90, 0x90, 0x90, 0x90,
		0xc3,
		0x0f, 0x84, 0x80, 0x03, 0x00, 0x00,
		0xcc, 0xcc, 0xcc,
	}

	ret[3] = byte(off1)
	ret[4] = byte(off1 >> 8)
	ret[5] = byte(off1 >> 16)
	ret[6] = byte(off1 >> 24)

	ret[20] = byte(off2)
	ret[21] = byte(off2 >> 8)
	ret[22] = byte(off2 >> 16)
	ret[23] = byte(off2 >> 24)

	fc1 := makeSliceFromPointer(target, len(ret))
	fc2 := makeSliceFromPointer(trampoline, len(ret))

	assert.Equal(t, ret[:8], fc2[:8])
	assert.Equal(t, byte(0xe9), fc2[8])
	assert.Equal(t, ret[8:], fc1[5:len(ret)-3])

	code2 := []byte{
		0x90, 0x90, 0x90, 0x90,
		0x74, 0x04,
		0x90, 0x90, 0x90, 0x90,
		0x90, 0x90, 0x90, 0x90, 0x90,
		0xc3, 0xcc, 0x90,
	}

	err2 := UnHook(foo_for_inplace_fix)
	assert.Nil(t, err2)

	msg2 := foo_for_inplace_fix_trampoline("txt")
	assert.Equal(t, "txtxxx3", msg2)

	msg3 := foo_for_inplace_fix_replace("txt2")
	assert.Equal(t, "txt2xxx2", msg3)

	CopyInstruction(target, code2)

	fsz, _ := GetFuncSizeByGuess(GetArchMode(), target, false)
	assert.Equal(t, len(code2)-1, int(fsz))
}

//go:noinline
func foo_short_call(a int) (int, error) {
	//fmt.Printf("calling short call origin func\n")
	return 3 + foo_short_call2(a), nil
}

//go:noinline
func foo_short_call2(a int) int {
	fmt.Printf("in short call2\n")
	return 3 + a
}

//go:noinline
func foo_short_call_replace(a int) (int, error) {
	fmt.Printf("calling short call replace func\n")
	r, _ := foo_short_call_trampoline(a)
	return a + 1000 + r, nil
}

func dummy_delimiter(id string) string {
	for {
		fmt.Printf("calling victim trampoline")
		if id == "miliao" {
			return "done"
		}
		break
	}

	ret := "miliao"
	ret += foo_for_inplace_fix("test")
	ret += foo_for_inplace_fix("test")
	ret += foo_for_inplace_fix("test")
	ret += foo_for_inplace_fix("test")

	fmt.Printf("len1:%d\n", len(id))
	fmt.Printf("len2:%d\n", len(ret))

	ret += foo_for_inplace_fix_delimiter(id)

	return id + ret
}

//go:noinline
func foo_short_call_trampoline(a int) (int, error) {
	for {
		fmt.Printf("printing a:%d\n", a)
		a++
		if a > 233 {
			fmt.Printf("done printing a:%d\n", a)
			break
		}
	}

	dummy_delimiter("miliao")

	return a + 233, nil
}

func TestShortCall(t *testing.T) {
	r, _ := foo_short_call(32)
	assert.Equal(t, 38, r)

	addr := GetFuncAddr(foo_short_call)
	sz1 := GetFuncInstSize(foo_short_call)
	addr2 := addr + uintptr(sz1)
	fmt.Printf("start hook real short call func, start:%x, end:%x\n", addr, addr2)

	err := Hook(foo_short_call, foo_short_call_replace, foo_short_call_trampoline)
	assert.Nil(t, err)

	r1, _ := foo_short_call(22)
	assert.Equal(t, 1050, r1)

	UnHook(foo_short_call)

	r2, _ := foo_short_call(32)
	assert.Equal(t, 38, r2)

	code := make([]byte, 0, sz1)
	for i := 0; i < int(sz1); i++ {
		code = append(code, 0x90)
	}

	code1 := []byte{0xeb, 0x4}
	code2 := []byte{0xeb, 0x5}

	copy(code, code1)
	copy(code[2:], code2)

	ret := sz1 - 5
	jmp1 := sz1 - 4
	jmp2 := sz1 - 2

	if sz1 > 0x7f {
		ret = 0x70 - 5
		jmp1 = 0x70 - 4
		jmp2 = 0x70 - 2
	}

	code[ret] = byte(0xc3)

	code3 := []byte{0xeb, byte(-jmp1 - 2)}
	code4 := []byte{0xeb, byte(-jmp2 - 2)}

	copy(code[jmp1:], code3)
	copy(code[jmp2:], code4)

	assert.Equal(t, code[:4], append(code1, code2...))

	CopyInstruction(addr, code)

	err = Hook(foo_short_call, foo_short_call_replace, foo_short_call_trampoline)
	assert.Nil(t, err)

	fmt.Printf("fix code for foo_short_call:\n%s\n", ShowDebugInfo())

	foo_short_call(22)

	addr3 := addr2 + uintptr(2)
	fc := runtime.FuncForPC(addr3)

	assert.NotNil(t, fc)

	fmt.Printf("func name get from addr beyond scope:%s\n", fc.Name())
	assert.Equal(t, addr, fc.Entry())

	f, l := fc.FileLine(addr2 + uintptr(3))
	assert.Equal(t, 0, l)
	assert.Equal(t, "?", f)
	fmt.Printf("file:%s, line:%d\n", f, l)
}
