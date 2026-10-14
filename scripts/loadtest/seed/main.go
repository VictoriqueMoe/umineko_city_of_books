package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"umineko_city_of_books/internal/config"
	appdb "umineko_city_of_books/internal/db"
)

const (
	defaultPassword = "loadtest-password"
)

var (
	databaseNamePattern = regexp.MustCompile(`^[a-z0-9_]+$`)
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

func run() error {
	dbName := flag.String("db", "umineko_loadtest", "database to drop, recreate and seed; must contain \"loadtest\"")
	out := flag.String("out", "scripts/loadtest/users.json", "where to write the seeded users and their session tokens")
	users := flag.Int("users", 1000, "number of users")
	posts := flag.Int("posts", 0, "number of posts (default 3 per user)")
	likes := flag.Int("likes", 0, "number of post likes (default 8 per post)")
	comments := flag.Int("comments", 0, "number of post comments (default 2 per post)")
	follows := flag.Int("follows", 20, "accounts each user follows")
	roomsPerUser := flag.Int("rooms-per-user", 3, "chat rooms each user belongs to")
	roomSize := flag.Int("room-size", 80, "members per chat room, below the 100 member cap")
	messages := flag.Int("messages", 40, "seeded messages per chat room")
	flag.Parse()

	cfg := seedConfig{
		Users:          *users,
		Posts:          *posts,
		Likes:          *likes,
		Comments:       *comments,
		FollowsPerUser: min(*follows, *users-1),
		RoomsPerUser:   *roomsPerUser,
		RoomSize:       *roomSize,
		MessagesEach:   *messages,
		Password:       defaultPassword,
	}
	if cfg.Posts == 0 {
		cfg.Posts = cfg.Users * 3
	}
	if cfg.Likes == 0 {
		cfg.Likes = cfg.Posts * 8
	}
	if cfg.Comments == 0 {
		cfg.Comments = cfg.Posts * 2
	}

	if cfg.Users < 2 || cfg.RoomsPerUser < 1 || cfg.RoomSize < 2 {
		return fmt.Errorf("need at least 2 users, 1 room per user and 2 members per room")
	}

	baseDSN := config.Cfg.PostgresDSN()
	if err := validateTarget(*dbName, databaseOf(baseDSN)); err != nil {
		return err
	}

	adminDSN, err := withDatabase(baseDSN, "postgres")
	if err != nil {
		return err
	}

	targetDSN, err := withDatabase(baseDSN, *dbName)
	if err != nil {
		return err
	}

	ctx := context.Background()

	fmt.Printf("recreating database %s\n", *dbName)
	if err := recreateDatabase(ctx, adminDSN, *dbName); err != nil {
		return err
	}

	database, err := appdb.Open(targetDSN)
	if err != nil {
		return fmt.Errorf("open %s: %w", *dbName, err)
	}
	defer database.Close()

	fmt.Println("applying migrations")
	if err := appdb.Migrate(database); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	if err := appdb.SeedContent(database); err != nil {
		return fmt.Errorf("seed content: %w", err)
	}

	if _, err := database.ExecContext(ctx, `
		INSERT INTO site_settings (key, value) VALUES ($1, 'warn')
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
		string(config.SettingLogLevel.Key)); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}

	fmt.Printf("seeding %d users, %d posts, %d likes, %d comments\n", cfg.Users, cfg.Posts, cfg.Likes, cfg.Comments)
	seeded, err := seed(ctx, database, cfg)
	if err != nil {
		return err
	}

	body, err := json.Marshal(seeded)
	if err != nil {
		return fmt.Errorf("encode users: %w", err)
	}

	if err := os.WriteFile(*out, body, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", *out, err)
	}

	fmt.Printf("wrote %d users with session tokens to %s (password for all of them: %s)\n", len(seeded), *out, defaultPassword)
	fmt.Printf("start the app against it with POSTGRES_DB=%s, or DATABASE_URL pointing at it if your .env sets DATABASE_URL\n", *dbName)

	return nil
}

func validateTarget(target string, configured string) error {
	if !databaseNamePattern.MatchString(target) {
		return fmt.Errorf("database name %q may only contain lower-case letters, digits and underscores", target)
	}

	if !strings.Contains(target, "loadtest") {
		return fmt.Errorf("refusing to drop %q: the name must contain \"loadtest\"", target)
	}

	if target == configured {
		return fmt.Errorf("refusing to drop %q: it is the database the app is configured to use", target)
	}

	return nil
}

func databaseOf(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return ""
	}

	return strings.TrimPrefix(u.Path, "/")
}

func withDatabase(dsn string, name string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("parse database url: %w", err)
	}

	u.Path = "/" + name

	return u.String(), nil
}

func recreateDatabase(ctx context.Context, adminDSN string, name string) error {
	admin, err := appdb.Open(adminDSN)
	if err != nil {
		return fmt.Errorf("open admin database: %w", err)
	}
	defer admin.Close()

	if _, err := admin.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", name)); err != nil {
		return fmt.Errorf("drop %s: %w", name, err)
	}

	if _, err := admin.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", name)); err != nil {
		return fmt.Errorf("create %s: %w", name, err)
	}

	return nil
}
