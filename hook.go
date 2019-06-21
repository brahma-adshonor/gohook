package gohook

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"
	"bytes"
)

type HookInfo struct {
	Mode        int
	Info        *CodeInfo
	Target      reflect.Value
	Replacement reflect.Value
	Trampoline  reflect.Value
}

var (
	archMode = 64
	g_all    = make(map[uintptr]HookInfo)
)

func init() {
	sz := unsafe.Sizeof(uintptr(0))
	if sz == 4 {
		archMode = 32
	}
}

func GetArchMode() int {
	return archMode
}

func ShowDebugInfo() string {
	buff := bytes.NewBuffer(make([]byte, 0, 256))
	for k,v := range g_all {
		s := fmt.Sprintf("hook function at addr:%x, how:%s, num of instruction fixed:%d\n", k, v.Info.How, len(v.Info.Fix))

		buff.WriteString(s)
		for _, f := range v.Info.Fix {
			s = fmt.Sprintf("==@%08x    new inst:", f.Addr)
			buff.WriteString(s)
			for _, c := range f.Code {
				s = fmt.Sprintf("%02x ", c)
				buff.WriteString(s)
			}
			s = fmt.Sprintf("\n")
			buff.WriteString(s)
		}
	}

	return string(buff.Bytes())
}

func Hook(target, replacement, trampoline interface{}) error {
	t := reflect.ValueOf(target)
	r := reflect.ValueOf(replacement)
	t2 := reflect.ValueOf(trampoline)
	return doHook(archMode, t, r, t2)
}

func UnHook(target interface{}) error {
	t := reflect.ValueOf(target)
	return doUnHook(t.Pointer())
}

func HookMethod(instance interface{}, method string, replacement, trampoline interface{}) error {
	target := reflect.TypeOf(instance)
	m, ok := target.MethodByName(method)
	if !ok {
		panic(fmt.Sprintf("unknown method %s", method))
	}
	r := reflect.ValueOf(replacement)
	t := reflect.ValueOf(trampoline)
	return doHook(archMode, m.Func, r, t)
}

func UnHookMethod(instance interface{}, methodName string) error {
	target := reflect.TypeOf(instance)
	m, ok := target.MethodByName(methodName)
	if !ok {
		return errors.New(fmt.Sprintf("unknown method %s", methodName))
	}

	return UnHook(m.Func.Interface())
}

func doUnHook(target uintptr) error {
	info, ok := g_all[target]
	if !ok {
		return errors.New("target not exist")
	}

	CopyInstruction(target, info.Info.Origin)

	if info.Info.How == "fix" {
		for _, v := range info.Info.Fix {
			CopyInstruction(v.Addr, v.Code)
		}
	}

	if info.Trampoline.IsValid() {
		CopyInstruction(info.Trampoline.Pointer(), info.Info.TrampolineOrig)
	}

	delete(g_all, target)

	return nil
}

func doHook(mode int, target, replacement, trampoline reflect.Value) error {
	if target.Kind() != reflect.Func {
		panic("target has to be a Func")
	}

	if replacement.Kind() != reflect.Func {
		panic("replacement has to be a Func")
	}

	if target.Type() != replacement.Type() {
		panic(fmt.Sprintf("target and replacement have to have the same type %s != %s", target.Type(), replacement.Type()))
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

	doUnHook(target.Pointer())

	info, err := hookFunction(mode, target.Pointer(), replacement.Pointer(), tp)
	if err != nil {
		return err
	}

	g_all[target.Pointer()] = HookInfo{Mode: mode, Info: info, Target: target, Replacement: replacement, Trampoline: trampoline}

	return nil
}
