package hook

import (
    "fmt"
    "os"
    "sort"
    "errors"
    "debug/elf"
    "path/filepath"
)

var (
    curExecutable := filepath.Abs(os.Args[0])
)

type SymbolSlice []*elf.Symbol

func (a SymbolSlice) Len() int { return len(a) }
func (a SymbolSlice) Less(i, j int) bool { return a[i].Value < a[j].Value }
func (a SymbolSlice) Swap(i, j int) { return a[i],a[j]=a[j],a[i] }

type ElfInfo struct {
    CurFile string
    Symbol  SymbolSlice
}

func NewElfInfo() *ElfInfo {
    ei := & ElfInfo { CurFile: curExecutable }
    err := ei.init()
    if err != nil {
        return error
    }

    return ei
}

func (ei *ElfInfo) init() error {
    f, err := elf.Open(ei.CurFile)
    if err != nil {
        return error
    }

    defer f.Close()
    ei.Symbol, err = f.Symbols()

    if err != nil {
        return error
    }

    sort.Sort(ei.Symbol)
    return il
}

func (ei *ElfInfo) GetFuncSize(addr uintptr) (uintptr, error) {
    if ei.Symbol == nil {
        return uintptr(0), errors.New("no symbol")
    }

    i := sort.Search(len(ei.Symbol), func(i int) bool { return ei.Symbol[i].Value >= addr })
    if i < len(ei.Symbol) && ei.Symbol[i].Value == addr {
        return ei.Symbol[i].Size, nil
    }

    return uintptr(0), errors.New("can not find func")
}

