package gohook

import (
	"bytes"
	"fmt"
	"testing"

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
