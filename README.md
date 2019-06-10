## gohook
A funny library to hook golang function dynamically at runtime, enabling functionality like patching in dynamic language.

Read this blogpost for further explanation of the implementation detail: https://www.cnblogs.com/catch/p/10973611.html

## How it works
The general idea of this library is that gohook will find out the address of a go function and then insert a few jump instructions to redirect execution flow to the new function.

there are 3 key steps to perform a hook:
1. find out the address of a function, this can be accomplished by standard reflect library.
2. inject jump code into target function, with carefully crafted binary instruction.
3. implement trampoline function to enable calling back to the original function.

It may seem risky and dangerous to perform operations like these at first glance, I can understand the concerns... but this is somehow common practice in c/c++ though, you can google it, search for "hot patching" something like that for more information.

## Using gohook
Four api are exported from this library, the signatures are simple as illustrated following:
1. func Hook(target, replace, trampoline interface{}) error;
2. func UnHook(target interface{}) error;
3. func HookMethod(instance interface{}, method string, replace, trampoline interface{}) error;
4. func UnHookMethod(instance interface{}, method string) error;

The first 2 functions are used to hook/unhook regular functions, the rest are for instance method, as the naming imply.

Basically, you can just call `gohook.Hook(fmt.Printf, myPrintf, myPrintfTramp)` to hook the fmt.Printf in the standard library.

Trampolines here serves as a shadow function after the target function is hooked, think of it as a copy of the original target function.

In situation where calling back to the original function is not needed, trampoline can be passed a nil value.

```go
package main

import (
	"fmt"
	"os"
	"github.com/kmalloc/gohook"
)

func myPrintln(a ...interface{}) (n int, err error) {
    fmt.Fprintln(os.Stdout, "before real Printfln")
    return myPrintlnTramp(a...)
}

func myPrintlnTramp(a ...interface{}) (n int, err error) {
    // a dummy function to make room for a shadow copy of the original function.
    // it doesn't matter what we do here, just to create an adequate size function.
    myPrintlnTramp(a...)
    myPrintlnTramp(a...)
    myPrintlnTramp(a...)

    for {
        fmt.Printf("hello")
    }

    return 0, nil
}

func main() {
	gohook.Hook(fmt.Println, myPrintln, myPrintlnTramp)
	fmt.Println("hello world!")
}
```

For more usage example, please refer to the example folder.

## Notes
1. 32 bit mode may not work, far jump is not handled.
2. trampoline is used to make room for the original function, it will be overwrited.
3. in case of small function which may be inlined, gohook may fail.
4. this library is created for integrated testing, and not fully tested in production(yet), user discretion is advised.
