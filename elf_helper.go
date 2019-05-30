package hook

import (
	"debug/elf"
	"errors"
	"os"
	"path/filepath"
	"sort"
)

var (
	curExecutable, _ = filepath.Abs(os.Args[0])
)

type SymbolSlice []elf.Symbol

func (a SymbolSlice) Len() int           { return len(a) }
func (a SymbolSlice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SymbolSlice) Less(i, j int) bool { return a[i].Value < a[j].Value }

type ElfInfo struct {
	CurFile string
	Symbol  SymbolSlice
}

func NewElfInfo() (*ElfInfo, error) {
	ei := &ElfInfo{CurFile: curExecutable}
	err := ei.init()
	if err != nil {
		return nil, err
	}

	return ei, nil
}

func (ei *ElfInfo) init() error {
	f, err := elf.Open(ei.CurFile)
	if err != nil {
		return err
	}

	defer f.Close()

	var sym []elf.Symbol
	sym, err = f.Symbols()
	ei.Symbol = SymbolSlice(sym)

	if err != nil {
		return err
	}

	sort.Sort(ei.Symbol)
	return nil
}

func (ei *ElfInfo) GetFuncSize(addr uintptr) (uint32, error) {
	if ei.Symbol == nil {
		return 0, errors.New("no symbol")
	}

	i := sort.Search(len(ei.Symbol), func(i int) bool { return ei.Symbol[i].Value >= uint64(addr) })
	if i < len(ei.Symbol) && ei.Symbol[i].Value == uint64(addr) {
		return uint32(ei.Symbol[i].Size), nil
	}

	return 0, errors.New("can not find func")
}
