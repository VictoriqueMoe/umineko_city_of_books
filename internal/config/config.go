package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type (
	Config struct {
		Postgres    PostgresConfig
		DatabaseURL string
		GiphyAPIKey string
	}

	PostgresConfig struct {
		Host     string
		Port     string
		User     string
		Password string
		DB       string
		SSLMode  string
	}

	SettingType int

	SiteSettingKey string

	SiteSettingDef struct {
		Key     SiteSettingKey
		Default string
		Type    SettingType
	}
)

func (c Config) PostgresDSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.Postgres.User, c.Postgres.Password),
		Host:     c.Postgres.Host + ":" + c.Postgres.Port,
		Path:     "/" + c.Postgres.DB,
		RawQuery: "sslmode=" + url.QueryEscape(c.Postgres.SSLMode),
	}
	return u.String()
}

const (
	TypeString SettingType = iota
	TypeBool
	TypeInt
)

var (
	Cfg Config

	Version = "dev"

	SettingUploadDir               = &SiteSettingDef{"upload_dir", "uploads", TypeString}
	SettingBaseURL                 = &SiteSettingDef{"base_url", "http://localhost:4323", TypeString}
	SettingLogLevel                = &SiteSettingDef{"log_level", "info", TypeString}
	SettingSentryDSN               = &SiteSettingDef{"sentry_dsn", "", TypeString}
	SettingOTLPEndpoint            = &SiteSettingDef{"otlp_endpoint", "", TypeString}
	SettingPyroscopeURL            = &SiteSettingDef{"pyroscope_url", "", TypeString}
	SettingMaxBodySize             = &SiteSettingDef{"max_body_size", "52428800", TypeInt}
	SettingMaxImageSize            = &SiteSettingDef{"max_image_size", "10485760", TypeInt}
	SettingMaxVideoSize            = &SiteSettingDef{"max_video_size", "104857600", TypeInt}
	SettingMaxGeneralSize          = &SiteSettingDef{"max_general_size", "52428800", TypeInt}
	SettingRegistrationType        = &SiteSettingDef{"registration_type", "open", TypeString}
	SettingMaintenanceMode         = &SiteSettingDef{"maintenance_mode", "false", TypeBool}
	SettingMaintenanceTitle        = &SiteSettingDef{"maintenance_title", "", TypeString}
	SettingMaintenanceMessage      = &SiteSettingDef{"maintenance_message", "", TypeString}
	SettingSiteName                = &SiteSettingDef{"site_name", "Umineko City of Books", TypeString}
	SettingSiteDescription         = &SiteSettingDef{"site_description", "", TypeString}
	SettingAnnouncementBanner      = &SiteSettingDef{"announcement_banner", "", TypeString}
	SettingMaxTheoriesPerDay       = &SiteSettingDef{"max_theories_per_day", "0", TypeInt}
	SettingMaxResponsesPerDay      = &SiteSettingDef{"max_responses_per_day", "0", TypeInt}
	SettingMaxPostsPerDay          = &SiteSettingDef{"max_posts_per_day", "0", TypeInt}
	SettingMinPasswordLength       = &SiteSettingDef{"min_password_length", "8", TypeInt}
	SettingSessionDurationDays     = &SiteSettingDef{"session_duration_days", "30", TypeInt}
	SettingDefaultTheme            = &SiteSettingDef{"default_theme", "featherine", TypeString}
	SettingDMsEnabled              = &SiteSettingDef{"dms_enabled", "true", TypeBool}
	SettingTurnstileEnabled        = &SiteSettingDef{"turnstile_enabled", "false", TypeBool}
	SettingTurnstileSiteKey        = &SiteSettingDef{"turnstile_site_key", "", TypeString}
	SettingTurnstileSecretKey      = &SiteSettingDef{"turnstile_secret_key", "", TypeString}
	SettingRulesTheories           = &SiteSettingDef{"rules_theories", "", TypeString}
	SettingRulesTheoriesHigurashi  = &SiteSettingDef{"rules_theories_higurashi", "", TypeString}
	SettingRulesMysteries          = &SiteSettingDef{"rules_mysteries", "", TypeString}
	SettingRulesShips              = &SiteSettingDef{"rules_ships", "", TypeString}
	SettingRulesGameBoard          = &SiteSettingDef{"rules_game_board", "", TypeString}
	SettingRulesGameBoardUmineko   = &SiteSettingDef{"rules_game_board_umineko", "", TypeString}
	SettingRulesGameBoardHigurashi = &SiteSettingDef{"rules_game_board_higurashi", "", TypeString}
	SettingRulesGameBoardCiconia   = &SiteSettingDef{"rules_game_board_ciconia", "", TypeString}
	SettingRulesGameBoardHiganbana = &SiteSettingDef{"rules_game_board_higanbana", "", TypeString}
	SettingRulesGameBoardRoseguns  = &SiteSettingDef{"rules_game_board_roseguns", "", TypeString}
	SettingMaxArtPerDay            = &SiteSettingDef{"max_art_per_day", "0", TypeInt}
	SettingMaxJournalsPerDay       = &SiteSettingDef{"max_journals_per_day", "0", TypeInt}
	SettingMaxChatRoomMembers      = &SiteSettingDef{"max_chat_room_members", "100", TypeInt}
	SettingMaxChatRoomsPerDay      = &SiteSettingDef{"max_chat_rooms_per_day", "0", TypeInt}
	SettingRulesGallery            = &SiteSettingDef{"rules_gallery", "", TypeString}
	SettingRulesGalleryUmineko     = &SiteSettingDef{"rules_gallery_umineko", "", TypeString}
	SettingRulesGalleryHigurashi   = &SiteSettingDef{"rules_gallery_higurashi", "", TypeString}
	SettingRulesGalleryCiconia     = &SiteSettingDef{"rules_gallery_ciconia", "", TypeString}
	SettingRulesFanfiction         = &SiteSettingDef{"rules_fanfiction", "", TypeString}
	SettingRulesJournals           = &SiteSettingDef{"rules_journals", "", TypeString}
	SettingRulesSuggestions        = &SiteSettingDef{"rules_suggestions", "", TypeString}
	SettingRulesChatRooms          = &SiteSettingDef{"rules_chat_rooms", "", TypeString}
	SettingSMTPHost                = &SiteSettingDef{"smtp_host", "", TypeString}
	SettingSMTPPort                = &SiteSettingDef{"smtp_port", "25", TypeInt}
	SettingSMTPFrom                = &SiteSettingDef{"smtp_from", "", TypeString}
	SettingSMTPUsername            = &SiteSettingDef{"smtp_username", "", TypeString}
	SettingSMTPPassword            = &SiteSettingDef{"smtp_password", "", TypeString}

	AllSiteSettings = []*SiteSettingDef{
		SettingUploadDir,
		SettingBaseURL,
		SettingLogLevel,
		SettingSentryDSN,
		SettingOTLPEndpoint,
		SettingPyroscopeURL,
		SettingMaxBodySize,
		SettingMaxImageSize,
		SettingMaxVideoSize,
		SettingMaxGeneralSize,
		SettingRegistrationType,
		SettingMaintenanceMode,
		SettingMaintenanceTitle,
		SettingMaintenanceMessage,
		SettingSiteName,
		SettingSiteDescription,
		SettingAnnouncementBanner,
		SettingMaxTheoriesPerDay,
		SettingMaxResponsesPerDay,
		SettingMaxPostsPerDay,
		SettingMinPasswordLength,
		SettingSessionDurationDays,
		SettingDefaultTheme,
		SettingDMsEnabled,
		SettingTurnstileEnabled,
		SettingTurnstileSiteKey,
		SettingTurnstileSecretKey,
		SettingRulesTheories,
		SettingRulesTheoriesHigurashi,
		SettingRulesMysteries,
		SettingRulesShips,
		SettingRulesGameBoard,
		SettingRulesGameBoardUmineko,
		SettingRulesGameBoardHigurashi,
		SettingRulesGameBoardCiconia,
		SettingRulesGameBoardHiganbana,
		SettingRulesGameBoardRoseguns,
		SettingMaxArtPerDay,
		SettingMaxJournalsPerDay,
		SettingMaxChatRoomMembers,
		SettingMaxChatRoomsPerDay,
		SettingRulesGallery,
		SettingRulesGalleryUmineko,
		SettingRulesGalleryHigurashi,
		SettingRulesGalleryCiconia,
		SettingRulesFanfiction,
		SettingRulesJournals,
		SettingRulesSuggestions,
		SettingRulesChatRooms,
		SettingSMTPHost,
		SettingSMTPPort,
		SettingSMTPFrom,
		SettingSMTPUsername,
		SettingSMTPPassword,
	}
)

func ValidateSettings(all map[SiteSettingKey]string) error {
	getInt := func(key SiteSettingKey) int {
		v, _ := strconv.Atoi(all[key])
		return v
	}

	maxBody := getInt(SettingMaxBodySize.Key)
	maxImage := getInt(SettingMaxImageSize.Key)
	maxVideo := getInt(SettingMaxVideoSize.Key)
	maxGeneral := getInt(SettingMaxGeneralSize.Key)
	minPassword := getInt(SettingMinPasswordLength.Key)
	sessionDays := getInt(SettingSessionDurationDays.Key)
	maxTheories := getInt(SettingMaxTheoriesPerDay.Key)
	maxResponses := getInt(SettingMaxResponsesPerDay.Key)

	if maxBody <= 0 {
		return fmt.Errorf("max body size must be greater than 0")
	}
	if maxImage <= 0 {
		return fmt.Errorf("max image size must be greater than 0")
	}
	if maxVideo <= 0 {
		return fmt.Errorf("max video size must be greater than 0")
	}
	if maxImage > maxBody {
		return fmt.Errorf("max image size (%d) cannot exceed max body size (%d)", maxImage, maxBody)
	}
	if maxVideo > maxBody {
		return fmt.Errorf("max video size (%d) cannot exceed max body size (%d)", maxVideo, maxBody)
	}
	if maxGeneral <= 0 {
		return fmt.Errorf("max general size must be greater than 0")
	}
	if maxGeneral > maxBody {
		return fmt.Errorf("max general size (%d) cannot exceed max body size (%d)", maxGeneral, maxBody)
	}
	if minPassword < 1 {
		return fmt.Errorf("minimum password length must be at least 1")
	}
	if sessionDays < 1 {
		return fmt.Errorf("session duration must be at least 1 day")
	}
	if maxTheories < 0 {
		return fmt.Errorf("max theories per day cannot be negative")
	}
	if maxResponses < 0 {
		return fmt.Errorf("max responses per day cannot be negative")
	}

	regType := all[SettingRegistrationType.Key]
	if regType != "open" && regType != "invite" && regType != "closed" {
		return fmt.Errorf("registration type must be 'open', 'invite', or 'closed'")
	}

	return nil
}

func init() {
	_ = godotenv.Load(".env", "postgres.env")

	pg := PostgresConfig{
		Host:     envOr("POSTGRES_HOST", "localhost"),
		Port:     envOr("POSTGRES_PORT", "5432"),
		User:     os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
		DB:       os.Getenv("POSTGRES_DB"),
		SSLMode:  envOr("POSTGRES_SSL_MODE", "disable"),
	}
	databaseURL := os.Getenv("DATABASE_URL")
	giphyKey := os.Getenv("GIPHY_API_KEY")

	Cfg = Config{
		Postgres:    pg,
		DatabaseURL: databaseURL,
		GiphyAPIKey: giphyKey,
	}

	for _, def := range AllSiteSettings {
		envKey := strings.ToUpper(string(def.Key))
		if v, ok := os.LookupEnv(envKey); ok {
			def.Default = v
		}
	}
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
