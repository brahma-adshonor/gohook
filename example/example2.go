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
