package hook

import (
	"fmt"
	"reflect"
	"errors"
)

type HookInfo struct {
	Mode        int
	Original []byte
	Target      reflect.Value
	Replacement reflect.Value
	Trampoline reflect.Value
}

var (
	g_all = make(map[uintptr]HookInfo)
)

func Hook(mode int, target, replacement, trampoline interface{}) {
	t := reflect.ValueOf(target)
	r := reflect.ValueOf(replacement)
	t2 := reflect.ValueOf(trampoline)
	doHook(mode, t, r, t2)
}

func UnHook(target interface{}) error {
	t := reflect.ValueOf(target)
	return doUnHook(t.Pointer())
}

func doUnHook(target uintptr) error{
	info, ok := g_all[target]
	if !ok {
		return errors.New("target not exist")
	}

	CopyInstruction(target, info.Original)
	delete(g_all, target)

	return nil
}


func HookInstanceMethod(mode int, target reflect.Type, method string, replacement, trampoline interface{}) {
	m, ok := target.MethodByName(method)
	if !ok {
		panic(fmt.Sprintf("unknown method %s", method))
	}
	r := reflect.ValueOf(replacement)
	t := reflect.ValueOf(trampoline)
	doHook(mode, m.Func, r, t)
}

func UnHookInstanceMethod(target reflect.Type, methodName string) error {
	m, ok := target.MethodByName(methodName)
	if !ok {
		return errors.New(fmt.Sprintf("unknown method %s", methodName))
	}

	return UnHook(m.Func)
}

func doHook(mode int, target, replacement, trampoline reflect.Value) {
	if target.Kind() != reflect.Func {
		panic("target has to be a Func")
	}

	if replacement.Kind() != reflect.Func {
		panic("replacement has to be a Func")
	}

	if target.Type() != replacement.Type() {
		panic(fmt.Sprintf("target and replacement have to have the same type %s != %s", target.Type(), trampoline.Type()))
	}

	tp := uintptr(0)
	if trampoline.IsValid() {
		if trampoline.Kind() != reflect.Func {
			panic("replacement has to be a Func")
		}

		if target.Type() != trampoline.Type() {
			panic(fmt.Sprintf("target and trampoline have to have the same type %s != %s", target.Type(), trampoline.Type()))
		}

		tp = trampoline.Pointer()
	}

	UnHook(target.Pointer())

	bytes := hookFunction(mode, target.Pointer(),replacement.Pointer(), tp)

	g_all[target.Pointer()] = HookInfo{Mode:mode, Original:bytes, Target:target, Replacement:replacement, Trampoline:trampoline}
}

