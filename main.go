package main

import (
	"fmt"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/utils"
)

func main() {
	logger.Init()

	logger.Log.Info().
		Str("db_path", config.Cfg.DBPath).
		Str("upload_dir", config.Cfg.UploadDir).
		Str("base_url", config.Cfg.BaseURL).
		Str("log_level", config.Cfg.LogLevel).
		Str("max_body_size", fmt.Sprintf("%d MB", config.Cfg.MaxBodySize/1024/1024)).
		Str("max_image_size", fmt.Sprintf("%d MB", config.Cfg.MaxImageSize/1024/1024)).
		Str("max_video_size", fmt.Sprintf("%d MB", config.Cfg.MaxVideoSize/1024/1024)).
		Msg("config loaded")

	app := initServer()

	logger.Log.Info().Str("addr", ":4323").Msg("starting server")
	utils.StartServerWithGracefulShutdown(app, ":4323")
}
