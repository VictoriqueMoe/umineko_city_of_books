package repotest

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	appdb "umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/db/dbtest"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

const (
	TestUserPassword = "password123"
	templateDBName   = "umineko_template"
)

var (
	hashOnce   sync.Once
	cachedHash string

	templateOnce sync.Once
	templateErr  error
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

func ensureTemplate() {
	templateOnce.Do(func() {
		if err := dbtest.EnsureRunning(); err != nil {
			templateErr = err
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		if err := dbtest.CreateDatabase(ctx, templateDBName); err != nil {
			templateErr = err
			return
		}

		templateDB, err := appdb.Open(dbtest.DSNFor(templateDBName))
		if err != nil {
			templateErr = fmt.Errorf("open template db: %w", err)
			return
		}

		if err := appdb.Migrate(templateDB); err != nil {
			_ = templateDB.Close()
			templateErr = fmt.Errorf("migrate template db: %w", err)
			return
		}

		if err := templateDB.Close(); err != nil {
			templateErr = fmt.Errorf("close template db: %w", err)
			return
		}

		if err := dbtest.MarkAsTemplate(ctx, templateDBName); err != nil {
			templateErr = err
			return
		}
	})
}

func CleanupTemplate() {
}

func NewDB(t *testing.T) *sql.DB {
	t.Helper()
	ensureTemplate()
	require.NoError(t, templateErr)
	return dbtest.NewDatabaseFromTemplate(t, templateDBName)
}

func NewRepos(t *testing.T) *repository.Repositories {
	t.Helper()
	return repository.New(NewDB(t))
}

type (
	UserOpt func(*userOpts)

	userOpts struct {
		username    string
		password    string
		displayName string
	}
)

func WithUsername(u string) UserOpt {
	return func(o *userOpts) {
		o.username = u
	}
}

func WithDisplayName(d string) UserOpt {
	return func(o *userOpts) {
		o.displayName = d
	}
}

func WithPassword(p string) UserOpt {
	return func(o *userOpts) {
		o.password = p
	}
}

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
		`INSERT INTO users (id, username, password_hash, display_name) VALUES ($1, $2, $3, $4)`,
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
