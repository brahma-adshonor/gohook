package main

import (
	"fmt"
	"hook"
    "bytes"
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

func hookFunc() {
    ret1 := foo1(23, "miliao for foo1 before hook")

    hook.Hook(64, foo1, foo2, foo3)

    fmt.Println("hook done")

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

func hookMethod() {
    buff := bytes.NewBufferString("abcd:")
	hook.HookMethod(64, buff, "WriteString", myBuffWriteString, myBuffWriteStringTramp)
    buff.WriteString("hook by miliao")
    fmt.Printf("value of buff:%s\n", buff.String())
}

func main() {
    fmt.Printf("start testing...\n")

    hookFunc()
    hookMethod()
}

