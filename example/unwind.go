package main

import (
	"fmt"
	"github.com/brahma-adshonor/gohook"
)

//go:noinline
func bad_func(k int) int {
	if k <= 0 {
		panic("bad func vomit\n")
	}

	return 23 + k
}

//go:noinline
func foo(k int) int {
	return k + bad_func(k)
}

//go:noinline
func goo(k int) int {
	k = -1000 + k
	return foo_trampoline(k)
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("recover from panic, err:%s\n", err)
		}
	}()

	gohook.Hook(foo, goo, foo_trampoline)
	fmt.Printf("hook info:%s\n", gohook.ShowDebugInfo())
	foo(-22)
}

//go:noinline
func foo_trampoline(k int) int {
	defer func(k int) { fmt.Printf("k:%d\n", k)}(k)

	k = -1000 + k
	return foo_trampoline(k)
}
