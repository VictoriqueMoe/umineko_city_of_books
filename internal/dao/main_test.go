package dao_test

import (
	"fmt"
	"os"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
)

func TestMain(m *testing.M) {
	fmt.Println("setup")
	code := m.Run()
	daotest.CleanupTemplate()
	os.Exit(code)
}
