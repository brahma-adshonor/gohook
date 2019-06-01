package hook

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
	"unsafe"
)

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

func myByteContain(a, b []byte) bool {
	fmt.Printf("calling fake bytes.Contain()\n")
	return false
}

func TestHook(t *testing.T) {
	fmt.Printf("start testing...\n")

	ret1 := foo1(23, "sval for foo1")
	assert.Equal(t, 65, ret1)

	Hook(64, foo1, foo2, foo3)

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
	Hook(64, bytes.Contains, myByteContain, nil)
	ret7 := bytes.Contains([]byte{1, 2, 3}, []byte{2, 3})
	assert.Equal(t, false, ret7)
	UnHook(bytes.Contains)
	ret8 := bytes.Contains([]byte{1, 2, 3}, []byte{2, 3})
	assert.Equal(t, true, ret8)
}

func myBuffLen(b *bytes.Buffer) int {
	fmt.Println("calling myBuffLen")
	return 233 + myBuffLenTramp(b)
}

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
	return 0, nil
}

func TestInstanceHook(t *testing.T) {
	buff1 := bytes.NewBufferString("abcd")
	assert.Equal(t, 4, buff1.Len())

	err1 := HookInstanceMethod(64, buff1, "Grow", myBuffGrow, nil)
	err2 := HookInstanceMethod(64, buff1, "Len", myBuffLen, myBuffLenTramp)

	assert.Nil(t, err1)
	assert.Nil(t, err2)

	assert.Equal(t, 4, buff1.Len()) // Len() is inlined
	buff1.Grow(233)                 // no grow
	assert.Equal(t, 4, buff1.Len()) // Len() is inlined

	err3 := HookInstanceMethod(64, buff1, "WriteString", myBuffWriteString, myBuffWriteStringTramp)
	assert.Nil(t, err3)

	sz1, _ := buff1.WriteString("miliao")
	assert.Equal(t, 1006, sz1)
	assert.Equal(t, 10, buff1.Len()) // Len() is inlined

	UnHookInstanceMethod(buff1, "WriteString")
	sz2, _ := buff1.WriteString("miliao")
	assert.Equal(t, 6, sz2)
	assert.Equal(t, 16, buff1.Len()) // Len() is inlined

	sz3, _ := myBuffWriteStringTramp(nil, "sssssss")
	assert.Equal(t, 0, sz3)
}

func TestGetInsLenGreaterThan(t *testing.T) {
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
	// jump from within patching erea to outside, negative fix
	c1 := []byte{0x75, 0x40} // jne 64

	l1, t1, r1 := FixOneInstruction(64, 10, 12, c1, 100, 8)

	assert.Equal(t, 2, l1)
	assert.Equal(t, FT_CondJmp, t1)
	assert.Equal(t, c1[0], r1[0])
	assert.Equal(t, int8(-26), int8(r1[1]))

	// jump from within patching erea to outside, positive fix
	l2, t2, r2 := FixOneInstruction(64, 10, 12, c1, 26, 8)

	assert.Equal(t, 2, l2)
	assert.Equal(t, FT_CondJmp, t2)
	assert.Equal(t, c1[0], r2[0])
	assert.Equal(t, int8(48), int8(r2[1]))

	//overflow test
	l3, t3, r3 := FixOneInstruction(64, 10, 12, c1, 1000, 8)

	assert.Equal(t, 2, l3)
	assert.Equal(t, FT_OVERFLOW, t3)
	assert.Equal(t, c1[0], r3[0])
	assert.Equal(t, c1[1], r3[1])

	// overflow test2
	c32 := []byte{0x75, 0x7e} // jne 0x7e
	l32, t32, r32 := FixOneInstruction(64, 30, 32, c32, 10, 8)

	assert.Equal(t, 2, l32)
	assert.Equal(t, FT_OVERFLOW, t32)
	assert.Equal(t, c32[0], r32[0])
	assert.Equal(t, c32[1], r32[1])

	// jump from outside patching erea to outside of patching erea
	l4, t4, r4 := FixOneInstruction(64, 10, 18, c1, 100, 4)

	assert.Equal(t, 2, l4)
	assert.Equal(t, FT_SKIP, t4)
	assert.Nil(t, r4)

	// jump from outside patching erea to within patching erea
	c2 := []byte{0x75, 0xe6} // jne -26
	l5, t5, r5 := FixOneInstruction(64, 10, 38, c2, 100, 8)

	assert.Equal(t, 2, l5)
	assert.Equal(t, FT_CondJmp, t5)
	assert.Equal(t, c2[0], r5[0])
	assert.Equal(t, 64, int(r5[1]))

	// jump within patching erea
	c3 := []byte{0x75, 0x06} // jne 6
	l6, t6, r6 := FixOneInstruction(64, 10, 12, c3, 100, 11)

	assert.Equal(t, 2, l6)
	assert.Equal(t, FT_SKIP, t6)
	assert.Nil(t, r6)

	// sign test, from outside to outside
	c4 := []byte{0x7c, 0xcd} // jne -51
	l7, t7, r7 := FixOneInstruction(64, 10, 83, c4, 1000, 10)

	assert.Equal(t, 2, l7)
	assert.Equal(t, FT_SKIP, t7)
	assert.Nil(t, r7)
}

func byteToInt32(d []byte) int32 {
	v := int32(uint32(d[0]) | (uint32(d[1]) << 8) | (uint32(d[2]) << 16) | (uint32(d[3]) << 24))
	return v
}

func TestFixOneInstructionForSixByteJmp(t *testing.T) {
	// jump from within patching erea to outside, negative fix
	c1 := []byte{0x0f, 0x8d, 0x10, 0x00, 0x00, 0x00} // jge 16

	l1, t1, r1 := FixOneInstruction(64, 20, 22, c1, 100, 8)
	assert.Equal(t, 6, l1)
	assert.Equal(t, FT_CondJmp, t1)
	assert.Equal(t, c1[0], r1[0])
	assert.Equal(t, c1[1], r1[1])

	assert.Equal(t, int32(-64), byteToInt32(r1[2:]))

	// jump from within patching erea to outside, positive fix
	c2 := []byte{0x0f, 0x8d, 0x40, 0x00, 0x00, 0x00} // jge 64

	l2, t2, r2 := FixOneInstruction(64, 2, 4, c2, 32, 9)
	assert.Equal(t, 6, l2)
	assert.Equal(t, FT_CondJmp, t2)
	assert.Equal(t, c2[0], r2[0])
	assert.Equal(t, c2[1], r2[1])

	assert.Equal(t, int32(34), byteToInt32(r2[2:]))

	// overflow test
	c3 := []byte{0x0f, 0x8d, 0xfe, 0xff, 0xff, 0x7f} // jge 64

	l3, t3, r3 := FixOneInstruction(64, 10000, 10004, c3, 100, 16)
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

	l4, t4, r4 := FixOneInstruction(64, 10, 33, c4, 22, 9)
	assert.Equal(t, 6, l4)
	assert.Equal(t, FT_SKIP, t4)
	assert.Nil(t, r4)

	// jump from outside patching erea to within patching erea
	c5 := []byte{0x0f, 0x85, 0xce, 0xff, 0xff, 0xff} // jne -50

	l5, t5, r5 := FixOneInstruction(64, 10, 60, c5, 1000, 9)
	assert.Equal(t, 6, l5)
	assert.Equal(t, FT_CondJmp, t5)
	assert.Equal(t, c5[0], r5[0])
	assert.Equal(t, c5[1], r5[1])

	assert.Equal(t, int32(940), byteToInt32(r5[2:]))

	// jump within patching erea
	c6 := []byte{0x0f, 0x85, 0x10, 0x00, 0x00, 0x00} // jne 16

	l6, t6, r6 := FixOneInstruction(64, 10, 12, c6, 1000, 30)
	assert.Equal(t, 6, l6)
	assert.Equal(t, FT_SKIP, t6)
	assert.Nil(t, r6)
}

func TestFixOneInstructionForFixByteJmp(t *testing.T) {
	// jump from within patching erea to outside, negative fix
	c1 := []byte{0xe9, 0x10, 0x00, 0x00, 0x00} // jmp 16

	l1, t1, r1 := FixOneInstruction(64, 20, 22, c1, 100, 8)
	assert.Equal(t, 5, l1)
	assert.Equal(t, FT_JMP, t1)
	assert.Equal(t, c1[0], r1[0])
	assert.Equal(t, int32(-64), byteToInt32(r1[1:]))

	// jump from within patching erea to outside, positive fix
	c2 := []byte{0xe9, 0x40, 0x00, 0x00, 0x00} // jmp 64

	l2, t2, r2 := FixOneInstruction(64, 2, 4, c2, 32, 9)
	assert.Equal(t, 5, l2)
	assert.Equal(t, FT_JMP, t2)
	assert.Equal(t, c2[0], r2[0])
	assert.Equal(t, int32(34), byteToInt32(r2[1:]))

	// overflow test
	c3 := []byte{0xe9, 0xfe, 0xff, 0xff, 0x7f} // jmp 64

	l3, t3, r3 := FixOneInstruction(64, 10000, 10004, c3, 100, 16)
	assert.Equal(t, 5, l3)
	assert.Equal(t, FT_OVERFLOW, t3)
	assert.Equal(t, c3[0], r3[0])
	assert.Equal(t, c3[1], r3[1])
	assert.Equal(t, c3[2], r3[2])
	assert.Equal(t, c3[3], r3[3])
	assert.Equal(t, c3[4], r3[4])

	// jump from outside patching erea to outside of patching erea
	c4 := []byte{0xe9, 0x40, 0x00, 0x00, 0x00} // jmp 64

	l4, t4, r4 := FixOneInstruction(64, 10, 33, c4, 22, 9)
	assert.Equal(t, 5, l4)
	assert.Equal(t, FT_SKIP, t4)
	assert.Nil(t, r4)

	// jump from outside patching erea to within patching erea
	c5 := []byte{0xe9, 0xce, 0xff, 0xff, 0xff} // jmp -50

	l5, t5, r5 := FixOneInstruction(64, 10, 60, c5, 1000, 9)
	assert.Equal(t, 5, l5)
	assert.Equal(t, FT_JMP, t5)
	assert.Equal(t, c5[0], r5[0])
	assert.Equal(t, int32(940), byteToInt32(r5[1:]))

	// jump within patching erea
	c6 := []byte{0xe9, 0x10, 0x00, 0x00, 0x00} // jmp 16

	l6, t6, r6 := FixOneInstruction(64, 10, 12, c6, 1000, 30)
	assert.Equal(t, 5, l6)
	assert.Equal(t, FT_SKIP, t6)
	assert.Nil(t, r6)

	// jump from outside to outside, sign test
	c7 := []byte{0xe8, 0xdc, 0xfb, 0xff, 0xff} // jmp -1060
	l7, t7, r7 := FixOneInstruction(64, 2000, 4100, c7, 10000, 30)
	assert.Equal(t, 5, l7)
	assert.Equal(t, FT_SKIP, t7)
	assert.Nil(t, r7)
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
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", a, e, f, g, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", a, e, f, g, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", a, e, f, g, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", a, e, f, g, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", a, e, f, g, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", a, e, f, g, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", a, e, f, g, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", a, e, f, g, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", a, e, f, g, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", a, e, f, g, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", a, e, f, g, 0x23)
	fmt.Printf("calling victim()(%s,%s,%s,%x):%dth\n", a, e, f, g, 0x23)

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

func TestStackGrowth(t *testing.T) {
	SetMinJmpCodeSize(64)
	defer SetMinJmpCodeSize(0)

	Hook(64, victim, victimReplace, victimTrampoline)

	ret := victim(0, 1000, 100000, "ab", "miliao", "see")

	assert.Equal(t, 42, ret)
}
