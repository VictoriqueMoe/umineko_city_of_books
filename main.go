package main

import (
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/utils"
)

func main() {
	logger.Init(config.SettingLogLevel.Default)
	defer logger.Shutdown()

	logger.Log.Info().
		Str("db_path", config.Cfg.DBPath).
		Msg("starting")

	app := initServer()

	logger.Log.Info().Str("addr", ":4323").Msg("starting server")
	utils.StartServerWithGracefulShutdown(app, ":4323")
}
