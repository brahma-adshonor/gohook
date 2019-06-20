package main

import (
	"fmt"
	"gohook"
)

type base struct {
	id int
}

type advance struct {
	base
	name string
}

type who interface {
	Id() int
}

//go:noinline
func (b *base) Id() int {
	return b.id
}

//go:noinline
func (a *advance) Name() string {
	return fmt.Sprintf("%d@%s", a.Id(), a.name)
}

func MyId(a *base) int {
	fmt.Printf("in fake MyId() base\n")
	return MyIdTrampoline(a) + 1000
}

//go:noinline
func MyIdTrampoline(a *base) int {
	fmt.Printf("abccccc")
	fmt.Printf("abccccc")
	fmt.Printf("abccccc")
	fmt.Printf("abccccc")

	for {
		if a != nil {
			fmt.Printf("bbbbbb")
		} else {
			fmt.Printf("cbbcbb")
		}
	}

	return 233
}

func MyId2(a *advance) int {
	fmt.Printf("in fake MyId() advance\n")
	return MyIdTrampoline2(a) + 5000
}

//go:noinline
func MyIdTrampoline2(a *advance) int {
	fmt.Printf("abccccc")
	fmt.Printf("abccccc")
	fmt.Printf("abccccc")
	fmt.Printf("abccccc")

	for {
		if a != nil {
			fmt.Printf("bbbbbb")
		} else {
			fmt.Printf("cbbcbb")
		}
	}

	return 233
}

func get_id_from(v who) int {
	return v.Id()
}

func main() {
	a := &advance{base:base{id:23},name:"miliao"}
	fmt.Printf("before hook advance, id:%d\n", a.Id())
	fmt.Printf("before hook advance, id from interface:%d\n", get_id_from(a))

	err := gohook.HookMethod(a, "Id", MyId2, MyIdTrampoline2)
	if err != nil {
		panic(fmt.Sprintf("Hook advance instance method failed:%s", err.Error()))
	}

	fmt.Printf("after hook advance, id:%d\n", a.Id())
	fmt.Printf("after hook advance, id from interface:%d\n", get_id_from(a))

	b := &base{id:333}
	err2 := gohook.HookMethod(b, "Id", MyId, MyIdTrampoline)
	if err2 != nil {
		panic(fmt.Sprintf("Hook base instance method failed:%s", err2.Error()))
	}

	fmt.Printf("after hook base, id:%d\n", a.Id())
	fmt.Printf("after hook base, id from interface:%d\n", get_id_from(a))

	fmt.Printf("debug info:\n%s\n", gohook.ShowDebugInfo())
}
