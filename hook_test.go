package hook

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
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

func TestAsm(t *testing.T) {
	fmt.Printf("start testing...\n")

	ret1 := foo1(23, "sval for foo1")
	assert.Equal(t, 65, ret1)

	Hook(64, foo1, foo2, foo3)

	ret2 := foo1(23, "sval for foo1")
	assert.Equal(t, 4342, ret2)
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

func TestFixOneInstructionForOneByteJmp(t *testing.T) {
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

	// jump from outside patching erea to whtin patching erea
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

	// TODO jmp/call
}

func TestFixOneInstructionForFourByteJmp(t *testing.T) {
	// jump from within patching erea to outside, negative fix
	c1 := []byte{0x0f, 0x8d, 0x10, 0x00, 0x00, 0x00} // jge 16

	l1, t1, r1 := FixOneInstruction(64, 20, 22, c1, 100, 8)
	assert.Equal(t, 6, l1)
	assert.Equal(t, FT_CondJmp, t1)
	assert.Equal(t, c1[0], r1[0])
	assert.Equal(t, c1[1], r1[1])

	assert.Equal(t, int32(-64), int32(uint32(r1[2])|(uint32(r1[3])<<8)|(uint32(r1[4])<<16)|(uint32(r1[5])<<24)))

	// jump from within patching erea to outside, positive fix
	c2 := []byte{0x0f, 0x8d, 0x40, 0x00, 0x00, 0x00} // jge 64

	l2, t2, r2 := FixOneInstruction(64, 2, 4, c2, 32, 9)
	assert.Equal(t, 6, l2)
	assert.Equal(t, FT_CondJmp, t2)
	assert.Equal(t, c2[0], r2[0])
	assert.Equal(t, c2[1], r2[1])

	assert.Equal(t, int32(34), int32(uint32(r2[2])|(uint32(r2[3])<<8)|(uint32(r2[4])<<16)|(uint32(r2[5])<<24)))

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
}
