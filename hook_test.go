package hook

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAsm(t *testing.T) {
	fmt.Printf("start testing...\n")
	RunTest()

	assert.Equal(t, 4, 4)
}
