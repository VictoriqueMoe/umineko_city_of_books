package dao_test

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	fmt.Println("setup")
	code := m.Run()
	os.Exit(code)
}
