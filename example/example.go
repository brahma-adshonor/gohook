package main

import (
	"bytes"
	"fmt"
	"github.com/brahma-adshonor/gohook"
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

func hookFunc() {
	ret1 := foo1(23, "miliao for foo1 before hook")

	err := gohook.Hook(foo1, foo2, foo3)

	fmt.Printf("hook done\n")
	if err != nil {
		fmt.Printf("err:%s\n", err.Error())
		return
	}

	ret2 := foo1(23, "miliao for foo1 after hook")

	fmt.Printf("r1:%d, r2:%d\n", ret1, ret2)
}

//  hook instance method
func myBuffWriteString(b *bytes.Buffer, s string) (int, error) {
	fmt.Printf("fake buffer WriteString func, s:%s\n", s)
	l, _ := myBuffWriteStringTramp(b, s)
	return 1000 + l, nil
}

func myBuffWriteStringTramp(b *bytes.Buffer, s string) (int, error) {
	fmt.Printf("fake buffer WriteString tramp, s:%s\n", s)
	return 0, nil
}

func myBuffLen(b *bytes.Buffer) int {
	fmt.Println("calling myBuffLen")
	return 233 + myBuffLenTramp(b)
}

//go:noinline
func myBuffLenTramp(b *bytes.Buffer) int {
	fmt.Println("calling myBuffLenTramp1")
	fmt.Println("calling myBuffLenTramp2")
	fmt.Println("calling myBuffLenTramp3")
	fmt.Println("calling myBuffLenTramp4")
	return 1000
}

func hookMethod() {
	buff := bytes.NewBufferString("abcd:")
	err1 := gohook.HookMethod(buff, "WriteString", myBuffWriteString, myBuffWriteStringTramp)
	if err1 != nil {
		fmt.Printf("hook WriteString() fail, err:%s\n", err1.Error())
		return
	}
	buff.WriteString("hook by miliao")
	fmt.Printf("value of buff:%s\n", buff.String())

	sz1 := buff.Len()

	fmt.Printf("try hook bytes.Buffer.Len()\n")
	err2 := gohook.HookMethod(buff, "Len", myBuffLen, myBuffLenTramp)
	if err2 != nil {
		fmt.Printf("hook Len() fail, err:%s\n", err2.Error())
		return
	}

	sz2 := buff.Len()
	sz3 := myBuffLenTramp(buff)

	gohook.UnHookMethod(buff, "Len")

	sz4 := myBuffLen(buff)
	fmt.Printf("old sz:%d, new sz:%d, copy func:%d, recover:%d\n", sz1, sz2, sz3, sz4)
}

func main() {
	fmt.Printf("start testing...\n")

	hookFunc()
	hookMethod()

	fmt.Printf("debug info:\n%s\n", gohook.ShowDebugInfo())
}
