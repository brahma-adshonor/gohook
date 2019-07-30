package gohook

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func victim(a, b, c int, e, f, g string) int {
	if a > 100 {
		return 42
	}

	var someBigStackArray [4096]byte // to occupy stack, don't let it escape
	for i := 0; i < len(someBigStackArray); i++ {
		someBigStackArray[i] = byte((a ^ b) & (i ^ c))
	}

	if (a % 2) != 0 {
		someBigStackArray[200] = 0xe9
	}

	fmt.Printf("calling real victim() (%s,%s,%s,%x):%dth\n", e, f, g, someBigStackArray[200], a)

	return victim(a+1, b-1, c-1, e, f, g)
}

func victimTrampoline(a, b, c int, e, f, g string) int {
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
	fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)

	for {
		if (a % 2) != 0 {
			fmt.Printf("calling victim()(%d,%s,%s,%x):%dth\n", a, e, f, c, 0x23)
		} else {
			a++
		}

		if a+b > 100 {
			break
		}

		buff := bytes.NewBufferString("something weird")
		fmt.Printf("len:%d\n", buff.Len())
	}

	return 1
}

func victimReplace(a, b, c int, e, f, g string) int {
	fmt.Printf("victimReplace sends its regard\n")
	ret := 0
	if a > 100 {
		ret = 100000
	}

	return ret + victimTrampoline(a, b, c, e, f, g)
}

func TestStackGrowth(t *testing.T) {
	SetMinJmpCodeSize(64)
	defer SetMinJmpCodeSize(0)

	ResetFuncPrologue()

	err := Hook(victim, victimReplace, victimTrampoline)
	assert.Nil(t, err)

	ret := victim(0, 1000, 100000, "ab", "miliao", "see")

	assert.Equal(t, 100042, ret)

	UnHook(victim)

	fmt.Printf("after unHook\n")
	victimReplace(98, 2, 3, "ab", "ef", "g")
}

func TestFuncSize(t *testing.T) {
	ResetFuncPrologue()

	addr1 := GetFuncAddr(victim)
	addr2 := GetFuncAddr(victimReplace)
	addr3 := GetFuncAddr(victimTrampoline)

	elf, err := NewElfInfo()
	hasElf := (err == nil)

	sz11, err11 := GetFuncSizeByGuess(GetArchMode(), addr1, true)
	assert.Nil(t, err11)

	if hasElf {
		sz1, err1 := elf.GetFuncSize(addr1)
		assert.Nil(t, err1)
		assert.Equal(t, sz1, sz11)
	} else {
		assert.True(t, sz11 > 0)
	}

	sz21, err21 := GetFuncSizeByGuess(GetArchMode(), addr2, true)
	assert.Nil(t, err21)

	if hasElf {
		sz2, err2 := elf.GetFuncSize(addr2)
		assert.Nil(t, err2)
		assert.Equal(t, sz2, sz21)
	}

	sz31, err31 := GetFuncSizeByGuess(GetArchMode(), addr3, true)
	assert.Nil(t, err31)

	if hasElf {
		sz3, err3 := elf.GetFuncSize(addr3)
		assert.Nil(t, err3)

		assert.Equal(t, sz3, sz31)
	}
}

func mySprintf(format string, a ...interface{}) string {
	addr1 := GetFuncAddr(victim)
	addr2 := GetFuncAddr(victimReplace)
	addr3 := GetFuncAddr(victimTrampoline)

	elf, err := NewElfInfo()
	fmt.Println("show:", elf, err)

	sz1, err1 := elf.GetFuncSize(addr1)
	fmt.Println("show:", sz1, err1)

	sz11, err11 := GetFuncSizeByGuess(GetArchMode(), addr1, false)
	fmt.Println("show:", sz11, err11)

	sz2, err2 := elf.GetFuncSize(addr2)
	fmt.Println("show:", sz2, err2)
	sz21, err21 := GetFuncSizeByGuess(GetArchMode(), addr2, false)
	fmt.Println("show:", sz21, err21)

	sz3, err3 := elf.GetFuncSize(addr3)
	fmt.Println("show:", sz3, err3)
	sz31, err31 := GetFuncSizeByGuess(GetArchMode(), addr3, false)
	fmt.Println("show:", sz31, err31)

	return ""
}

func TestCopyFunc(t *testing.T) {
	ResetFuncPrologue()

	addr := GetFuncAddr(mySprintf)
	sz := GetFuncInstSize(mySprintf)

	tp := makeSliceFromPointer(addr, int(sz))
	txt := make([]byte, int(sz))
	copy(txt, tp)

	fs := "some random text, from %d,%S,%T"
	s1 := fmt.Sprintf(fs, 233, "miliao test sprintf", addr)

	info := &CodeInfo{}
	origin, err := CopyFunction(true, fmt.Sprintf, mySprintf, info)

	assert.Nil(t, err)
	assert.Equal(t, len(txt), len(origin))
	assert.Equal(t, txt, origin)

	s2 := mySprintf(fs, 233, "miliao test sprintf", addr)

	assert.Equal(t, s1, s2)

	addr2 := GetFuncAddr(fmt.Sprintf)
	sz2, _ := GetFuncSizeByGuess(GetArchMode(), addr2, true)
	sz3, _ := GetFuncSizeByGuess(GetArchMode(), addr, true)

	assert.Equal(t, sz2, sz3)
}
