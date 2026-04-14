package repository_test

import (
	"fmt"
	"os"
	"testing"

	"umineko_city_of_books/internal/repository/repotest"
)

func TestMain(m *testing.M) {
	fmt.Println("setup")
	code := m.Run()
	repotest.CleanupTemplate()
	os.Exit(code)
}
