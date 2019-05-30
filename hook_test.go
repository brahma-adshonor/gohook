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
    c1 := []byte{0x64,0x48,0x8b,0x0c,0x25,0xf8}
    c2 := []byte{0x64,0x48,0x8b,0x0c,0x25,0xf8,0xff,0xff,0xff}

    r1 := GetInsLenGreaterThan(64, c1, len(c1) - 2)
    assert.Equal(t, 0, r1)
    r2 := GetInsLenGreaterThan(64, c2, len(c2) - 2)
    assert.Equal(t, len(c2), r2)
    r22 := GetInsLenGreaterThan(64, c2, len(c2))
    assert.Equal(t, len(c2), r22)
    r23 := GetInsLenGreaterThan(64, c2, len(c2) + 2)
    assert.Equal(t, 0, r23)

    c3 := []byte{0x64,0x48,0x8b,0x0c,0x25,0xf8,0xff,0xff,0xff,0x48,0x3b,0x41,0x10}
    r3 := GetInsLenGreaterThan(64, c3, len(c2) + 2)
    assert.Equal(t, len(c3), r3)
    r32 := GetInsLenGreaterThan(64, c3, len(c2) - 2)
    assert.Equal(t, len(c2), r32)
}

func TestFixOneInstruction(t *testing.T) {
    c1 := []byte{0x75,0x40}
    l1, t1, r1 := FixOneInstruction(64, 10, 12, c1, 100, 8)

    assert.Equal(t, 2, l1)
    assert.Equal(t, FT_CondJmp, t1)
    assert.Equal(t, c1[0], r1[0])
    assert.Equal(t, int8(-26), int8(r1[1]))

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
}

