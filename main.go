package main

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/controllers"
	"umineko_city_of_books/internal/db"
	appMiddleware "umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/profile"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/routes"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/theory"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/user"
	"umineko_city_of_books/internal/utils"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

var (
	//go:embed static/*
	staticFiles embed.FS
)

func main() {
	if err := os.MkdirAll(filepath.Join(config.Cfg.UploadDir, "avatars"), 0755); err != nil {
		log.Fatalf("failed to create uploads directory: %v", err)
	}

	database, err := db.Open(config.Cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	repos := repository.New(database)

	sessionMgr := session.NewManager(repos.Session)
	uploadService := upload.NewService()
	userService := user.NewService(repos.User)
	authService := auth.NewService(userService, sessionMgr)
	profileService := profile.NewService(repos.User, uploadService)
	theoryService := theory.NewService(repos.Theory)

	app := fiber.New()

	appMiddleware.Setup(app, config.Cfg.BaseURL)

	htmlBytes, _ := staticFiles.ReadFile("static/index.html")
	service := controllers.NewService(authService, profileService, theoryService, sessionMgr, string(htmlBytes))
	routes.PublicRoutes(service, app)

	app.Get("/uploads/*", static.New(uploadService.GetUploadDir()))

	staticFS, _ := fs.Sub(staticFiles, "static")
	app.Get("/*", func(ctx fiber.Ctx) error {
		path := ctx.Path()
		if strings.Contains(path, ".") {
			return static.New("", static.Config{
				FS: staticFS,
			})(ctx)
		}
		return ctx.Type("html").Send(htmlBytes)
	})

	utils.StartServerWithGracefulShutdown(app, ":4323")
}
