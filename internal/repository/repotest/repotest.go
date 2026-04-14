package repotest

import (
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	appdb "umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

const TestUserPassword = "password123"

var (
	hashOnce   sync.Once
	cachedHash string
)

func testPasswordHash(t *testing.T) string {
	t.Helper()
	hashOnce.Do(func() {
		h, err := bcrypt.GenerateFromPassword([]byte(TestUserPassword), bcrypt.MinCost)
		if err != nil {
			panic(err)
		}
		cachedHash = string(h)
	})
	return cachedHash
}

var (
	templateOnce sync.Once
	templatePath string
	templateErr  error
)

func buildTemplate() {
	dir, err := os.MkdirTemp("", "repotest-template-*")
	if err != nil {
		templateErr = err
		return
	}
	templatePath = filepath.Join(dir, "template.db")

	db, err := appdb.Open(templatePath)
	if err != nil {
		templateErr = err
		return
	}
	defer db.Close()

	if err := appdb.Migrate(db); err != nil {
		templateErr = err
		return
	}
}

func CleanupTemplate() {
	if templatePath == "" {
		return
	}
	_ = os.RemoveAll(filepath.Dir(templatePath))
}

func NewDB(t *testing.T) *sql.DB {
	t.Helper()
	templateOnce.Do(buildTemplate)
	require.NoError(t, templateErr)

	dbPath := filepath.Join(t.TempDir(), "test.db")
	require.NoError(t, copyFile(templatePath, dbPath))

	db, err := appdb.Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func NewRepos(t *testing.T) *repository.Repositories {
	t.Helper()
	return repository.New(NewDB(t))
}

type UserOpt func(*userOpts)

type userOpts struct {
	username    string
	password    string
	displayName string
}

func WithUsername(u string) UserOpt    { return func(o *userOpts) { o.username = u } }
func WithDisplayName(d string) UserOpt { return func(o *userOpts) { o.displayName = d } }
func WithPassword(p string) UserOpt    { return func(o *userOpts) { o.password = p } }

func CreateUser(t *testing.T, repos *repository.Repositories, opts ...UserOpt) *model.User {
	t.Helper()
	o := userOpts{
		username:    "user_" + uuid.New().String()[:8],
		password:    TestUserPassword,
		displayName: "Test User",
	}
	for _, opt := range opts {
		opt(&o)
	}
	if o.password != TestUserPassword {
		u, err := repos.User.Create(context.Background(), o.username, o.password, o.displayName)
		require.NoError(t, err)
		return u
	}

	id := uuid.New()
	_, err := repos.DB().ExecContext(
		context.Background(),
		`INSERT INTO users (id, username, password_hash, display_name) VALUES (?, ?, ?, ?)`,
		id, o.username, testPasswordHash(t), o.displayName,
	)
	require.NoError(t, err)
	return &model.User{ID: id, Username: o.username, DisplayName: o.displayName}
}

func CreateSession(t *testing.T, repos *repository.Repositories, userID uuid.UUID) string {
	t.Helper()
	token := uuid.NewString()
	require.NoError(t, repos.Session.Create(context.Background(), token, userID, time.Now().Add(time.Hour)))
	return token
}
