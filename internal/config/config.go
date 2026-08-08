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
		Postgres           PostgresConfig
		DatabaseURL        string
		GiphyAPIKey        string
		FCMCredentialsFile string
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

	EmailProvider string

	SiteSettingDef struct {
		Key     SiteSettingKey
		Default string
		Type    SettingType
		Secret  bool
	}
)

const (
	TypeString SettingType = iota
	TypeBool
	TypeInt
)

const (
	EmailProviderSMTP       EmailProvider = "smtp"
	EmailProviderCloudflare EmailProvider = "cloudflare"
)

const (
	SecretMask = "********"
)

var (
	Cfg Config

	Version = "dev"

	SettingUploadDir               = &SiteSettingDef{"upload_dir", "uploads", TypeString, false}
	SettingBaseURL                 = &SiteSettingDef{"base_url", "http://localhost:4323", TypeString, false}
	SettingLogLevel                = &SiteSettingDef{"log_level", "info", TypeString, false}
	SettingSentryDSN               = &SiteSettingDef{"sentry_dsn", "", TypeString, true}
	SettingOTLPEndpoint            = &SiteSettingDef{"otlp_endpoint", "", TypeString, false}
	SettingPyroscopeURL            = &SiteSettingDef{"pyroscope_url", "", TypeString, false}
	SettingMaxBodySize             = &SiteSettingDef{"max_body_size", "52428800", TypeInt, false}
	SettingMaxImageSize            = &SiteSettingDef{"max_image_size", "10485760", TypeInt, false}
	SettingMaxImagePixels          = &SiteSettingDef{"max_image_pixels", "100000000", TypeInt, false}
	SettingMaxVideoSize            = &SiteSettingDef{"max_video_size", "104857600", TypeInt, false}
	SettingMaxGeneralSize          = &SiteSettingDef{"max_general_size", "52428800", TypeInt, false}
	SettingRegistrationType        = &SiteSettingDef{"registration_type", "open", TypeString, false}
	SettingMaintenanceMode         = &SiteSettingDef{"maintenance_mode", "false", TypeBool, false}
	SettingMaintenanceTitle        = &SiteSettingDef{"maintenance_title", "", TypeString, false}
	SettingMaintenanceMessage      = &SiteSettingDef{"maintenance_message", "", TypeString, false}
	SettingSiteName                = &SiteSettingDef{"site_name", "When They Cry City of Books", TypeString, false}
	SettingSiteDescription         = &SiteSettingDef{"site_description", "", TypeString, false}
	SettingAnnouncementBanner      = &SiteSettingDef{"announcement_banner", "", TypeString, false}
	SettingMaxTheoriesPerDay       = &SiteSettingDef{"max_theories_per_day", "0", TypeInt, false}
	SettingMaxResponsesPerDay      = &SiteSettingDef{"max_responses_per_day", "0", TypeInt, false}
	SettingMaxPostsPerDay          = &SiteSettingDef{"max_posts_per_day", "0", TypeInt, false}
	SettingMinPasswordLength       = &SiteSettingDef{"min_password_length", "8", TypeInt, false}
	SettingSessionDurationDays     = &SiteSettingDef{"session_duration_days", "30", TypeInt, false}
	SettingDefaultTheme            = &SiteSettingDef{"default_theme", "featherine", TypeString, false}
	SettingDMsEnabled              = &SiteSettingDef{"dms_enabled", "true", TypeBool, false}
	SettingVoiceEnabled            = &SiteSettingDef{"voice_enabled", "false", TypeBool, false}
	SettingLiveKitURL              = &SiteSettingDef{"livekit_url", "", TypeString, false}
	SettingLiveKitAPIKey           = &SiteSettingDef{"livekit_api_key", "", TypeString, false}
	SettingLiveKitAPISecret        = &SiteSettingDef{"livekit_api_secret", "", TypeString, true}
	SettingHyperbeamAPIKey         = &SiteSettingDef{"hyperbeam_api_key", "", TypeString, true}
	SettingHyperbeamRegion         = &SiteSettingDef{"hyperbeam_region", "EU", TypeString, false}
	SettingStreamingEnabled        = &SiteSettingDef{"streaming_enabled", "false", TypeBool, false}
	SettingStreamMaxConcurrent     = &SiteSettingDef{"stream_max_concurrent", "3", TypeInt, false}
	SettingStreamHLSEnabled        = &SiteSettingDef{"stream_hls_enabled", "false", TypeBool, false}
	SettingStreamHLSOutputDir      = &SiteSettingDef{"stream_hls_output_dir", "/app/data/hls", TypeString, false}
	SettingTurnstileEnabled        = &SiteSettingDef{"turnstile_enabled", "false", TypeBool, false}
	SettingTurnstileSiteKey        = &SiteSettingDef{"turnstile_site_key", "", TypeString, false}
	SettingTurnstileSecretKey      = &SiteSettingDef{"turnstile_secret_key", "", TypeString, true}
	SettingRulesTheories           = &SiteSettingDef{"rules_theories", "", TypeString, false}
	SettingRulesTheoriesHigurashi  = &SiteSettingDef{"rules_theories_higurashi", "", TypeString, false}
	SettingRulesTheoriesCiconia    = &SiteSettingDef{"rules_theories_ciconia", "", TypeString, false}
	SettingRulesMysteries          = &SiteSettingDef{"rules_mysteries", "", TypeString, false}
	SettingRulesShips              = &SiteSettingDef{"rules_ships", "", TypeString, false}
	SettingRulesGameBoard          = &SiteSettingDef{"rules_game_board", "", TypeString, false}
	SettingRulesGameBoardUmineko   = &SiteSettingDef{"rules_game_board_umineko", "", TypeString, false}
	SettingRulesGameBoardHigurashi = &SiteSettingDef{"rules_game_board_higurashi", "", TypeString, false}
	SettingRulesGameBoardCiconia   = &SiteSettingDef{"rules_game_board_ciconia", "", TypeString, false}
	SettingRulesGameBoardHiganbana = &SiteSettingDef{"rules_game_board_higanbana", "", TypeString, false}
	SettingRulesGameBoardRoseguns  = &SiteSettingDef{"rules_game_board_roseguns", "", TypeString, false}
	SettingMaxArtPerDay            = &SiteSettingDef{"max_art_per_day", "0", TypeInt, false}
	SettingMaxJournalsPerDay       = &SiteSettingDef{"max_journals_per_day", "0", TypeInt, false}
	SettingMaxChatRoomMembers      = &SiteSettingDef{"max_chat_room_members", "100", TypeInt, false}
	SettingMaxChatRoomsPerDay      = &SiteSettingDef{"max_chat_rooms_per_day", "0", TypeInt, false}
	SettingRulesGallery            = &SiteSettingDef{"rules_gallery", "", TypeString, false}
	SettingRulesGalleryUmineko     = &SiteSettingDef{"rules_gallery_umineko", "", TypeString, false}
	SettingRulesGalleryHigurashi   = &SiteSettingDef{"rules_gallery_higurashi", "", TypeString, false}
	SettingRulesGalleryCiconia     = &SiteSettingDef{"rules_gallery_ciconia", "", TypeString, false}
	SettingRulesFanfiction         = &SiteSettingDef{"rules_fanfiction", "", TypeString, false}
	SettingRulesJournals           = &SiteSettingDef{"rules_journals", "", TypeString, false}
	SettingRulesSuggestions        = &SiteSettingDef{"rules_suggestions", "", TypeString, false}
	SettingRulesChatRooms          = &SiteSettingDef{"rules_chat_rooms", "", TypeString, false}
	SettingRulesPage               = &SiteSettingDef{"rules_page", "", TypeString, false}
	SettingRulesLanding            = &SiteSettingDef{"rules_landing", "", TypeString, false}
	SettingSMTPHost                = &SiteSettingDef{"smtp_host", "", TypeString, false}
	SettingSMTPPort                = &SiteSettingDef{"smtp_port", "25", TypeInt, false}
	SettingSMTPFrom                = &SiteSettingDef{"smtp_from", "", TypeString, false}
	SettingSMTPUsername            = &SiteSettingDef{"smtp_username", "", TypeString, false}
	SettingSMTPPassword            = &SiteSettingDef{"smtp_password", "", TypeString, true}
	SettingEmailProvider           = &SiteSettingDef{"email_provider", string(EmailProviderSMTP), TypeString, false}
	SettingCloudflareAccountID     = &SiteSettingDef{"cloudflare_account_id", "", TypeString, false}
	SettingCloudflareAPIToken      = &SiteSettingDef{"cloudflare_api_token", "", TypeString, true}
	SettingCloudflareEmailFrom     = &SiteSettingDef{"cloudflare_email_from", "", TypeString, false}
	SettingPushEnabled             = &SiteSettingDef{"push_enabled", "false", TypeBool, false}
	SettingAppLatestVersion        = &SiteSettingDef{"app_latest_version", "", TypeString, false}
	SettingAppDownloadURL          = &SiteSettingDef{"app_download_url", "", TypeString, false}
	SettingOGDefaultImage          = &SiteSettingDef{"og_default_image", "", TypeString, false}
	SettingValkeyURL               = &SiteSettingDef{"valkey_url", "", TypeString, true}

	SettingChatbotEnabled              = &SiteSettingDef{"chatbot_enabled", "false", TypeBool, false}
	SettingChatbotAPIKey               = &SiteSettingDef{"chatbot_api_key", "", TypeString, true}
	SettingChatbotAdminKey             = &SiteSettingDef{"chatbot_admin_key", "", TypeString, true}
	SettingChatbotModel                = &SiteSettingDef{"chatbot_model", "gpt-5.6-luna", TypeString, false}
	SettingChatbotReasoningEffort      = &SiteSettingDef{"chatbot_reasoning_effort", "low", TypeString, false}
	SettingChatbotVerbosity            = &SiteSettingDef{"chatbot_verbosity", "", TypeString, false}
	SettingChatbotMaxOutputTokens      = &SiteSettingDef{"chatbot_max_output_tokens", "2000", TypeInt, false}
	SettingChatbotContextMessages      = &SiteSettingDef{"chatbot_context_messages", "20", TypeInt, false}
	SettingChatbotMaxReplyChain        = &SiteSettingDef{"chatbot_max_reply_chain", "25", TypeInt, false}
	SettingChatbotRequirePermission    = &SiteSettingDef{"chatbot_require_permission", "false", TypeBool, false}
	SettingChatbotOptInRole            = &SiteSettingDef{"chatbot_opt_in_role", "", TypeString, false}
	SettingChatbotReplyCooldownSeconds = &SiteSettingDef{"chatbot_reply_cooldown_seconds", "20", TypeInt, false}
	SettingChatbotMaxRepliesPerUserDay = &SiteSettingDef{"chatbot_max_replies_per_user_per_day", "20", TypeInt, false}
	SettingChatbotMaxRepliesPerDay     = &SiteSettingDef{"chatbot_max_replies_per_day", "500", TypeInt, false}

	AllSiteSettings = []*SiteSettingDef{
		SettingUploadDir,
		SettingBaseURL,
		SettingLogLevel,
		SettingSentryDSN,
		SettingOTLPEndpoint,
		SettingPyroscopeURL,
		SettingMaxBodySize,
		SettingMaxImageSize,
		SettingMaxImagePixels,
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
		SettingVoiceEnabled,
		SettingLiveKitURL,
		SettingLiveKitAPIKey,
		SettingLiveKitAPISecret,
		SettingHyperbeamAPIKey,
		SettingHyperbeamRegion,
		SettingStreamingEnabled,
		SettingStreamMaxConcurrent,
		SettingStreamHLSEnabled,
		SettingStreamHLSOutputDir,
		SettingTurnstileEnabled,
		SettingTurnstileSiteKey,
		SettingTurnstileSecretKey,
		SettingRulesTheories,
		SettingRulesTheoriesHigurashi,
		SettingRulesTheoriesCiconia,
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
		SettingRulesPage,
		SettingRulesLanding,
		SettingSMTPHost,
		SettingSMTPPort,
		SettingSMTPFrom,
		SettingSMTPUsername,
		SettingSMTPPassword,
		SettingEmailProvider,
		SettingCloudflareAccountID,
		SettingCloudflareAPIToken,
		SettingCloudflareEmailFrom,
		SettingPushEnabled,
		SettingAppLatestVersion,
		SettingAppDownloadURL,
		SettingOGDefaultImage,
		SettingValkeyURL,
		SettingChatbotEnabled,
		SettingChatbotAPIKey,
		SettingChatbotAdminKey,
		SettingChatbotModel,
		SettingChatbotReasoningEffort,
		SettingChatbotVerbosity,
		SettingChatbotMaxOutputTokens,
		SettingChatbotContextMessages,
		SettingChatbotMaxReplyChain,
		SettingChatbotRequirePermission,
		SettingChatbotOptInRole,
		SettingChatbotReplyCooldownSeconds,
		SettingChatbotMaxRepliesPerUserDay,
		SettingChatbotMaxRepliesPerDay,
	}

	settingsByKey = indexSettings()
)

func indexSettings() map[SiteSettingKey]*SiteSettingDef {
	byKey := make(map[SiteSettingKey]*SiteSettingDef, len(AllSiteSettings))
	for _, def := range AllSiteSettings {
		byKey[def.Key] = def
	}

	return byKey
}

func SettingByKey(key SiteSettingKey) (*SiteSettingDef, bool) {
	def, ok := settingsByKey[key]

	return def, ok
}

func ValidateSettingValue(def *SiteSettingDef, value string) error {
	switch def.Type {
	case TypeInt:
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("%s must be a whole number", def.Key)
		}
	case TypeBool:
		if value != "true" && value != "false" {
			return fmt.Errorf("%s must be true or false", def.Key)
		}
	}

	return nil
}

func validChatbotVerbosity(value string) bool {
	switch strings.TrimSpace(value) {
	case "", "low", "medium", "high":
		return true
	}

	return false
}

func validateChatbotSettings(all map[SiteSettingKey]string) error {
	if all[SettingChatbotEnabled.Key] != "true" {
		return nil
	}

	if all[SettingChatbotRequirePermission.Key] == "true" && strings.TrimSpace(all[SettingChatbotOptInRole.Key]) == "" {
		return fmt.Errorf("restricting characters to a permission requires an opt-in role so members can opt in")
	}

	if all[SettingChatbotAPIKey.Key] == "" {
		return fmt.Errorf("the chatbot requires an OpenAI API key")
	}
	if all[SettingChatbotModel.Key] == "" {
		return fmt.Errorf("the chatbot requires a model")
	}

	switch all[SettingChatbotReasoningEffort.Key] {
	case "none", "low", "medium", "high", "xhigh", "max":
	default:
		return fmt.Errorf("reasoning effort must be one of none, low, medium, high, xhigh, max")
	}

	if !validChatbotVerbosity(all[SettingChatbotVerbosity.Key]) {
		return fmt.Errorf("verbosity must be low, medium, high, or left blank to use the provider default")
	}

	positive := []*SiteSettingDef{
		SettingChatbotMaxOutputTokens,
		SettingChatbotContextMessages,
		SettingChatbotMaxReplyChain,
	}
	for _, def := range positive {
		n, err := strconv.Atoi(all[def.Key])
		if err != nil || n <= 0 {
			return fmt.Errorf("%s must be a positive number", def.Key)
		}
	}

	nonNegative := []*SiteSettingDef{
		SettingChatbotReplyCooldownSeconds,
		SettingChatbotMaxRepliesPerUserDay,
		SettingChatbotMaxRepliesPerDay,
	}
	for _, def := range nonNegative {
		n, err := strconv.Atoi(all[def.Key])
		if err != nil || n < 0 {
			return fmt.Errorf("%s must be zero or a positive number", def.Key)
		}
	}

	return nil
}

func ValidateSettings(all map[SiteSettingKey]string) error {
	getInt := func(key SiteSettingKey) int {
		v, _ := strconv.Atoi(all[key])
		return v
	}

	maxBody := getInt(SettingMaxBodySize.Key)
	maxImage := getInt(SettingMaxImageSize.Key)
	maxImagePixels := getInt(SettingMaxImagePixels.Key)
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
	if maxImagePixels <= 0 {
		return fmt.Errorf("max image pixels must be greater than 0")
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

	ogImage := all[SettingOGDefaultImage.Key]
	if ogImage != "" {
		if !strings.HasPrefix(ogImage, "/uploads/") || !strings.HasSuffix(strings.ToLower(ogImage), ".jpg") {
			return fmt.Errorf("default embed image must be an uploaded .jpg file")
		}
	}

	if all[SettingVoiceEnabled.Key] == "true" {
		if all[SettingLiveKitURL.Key] == "" || all[SettingLiveKitAPIKey.Key] == "" || all[SettingLiveKitAPISecret.Key] == "" {
			return fmt.Errorf("voice chat requires LiveKit URL, API key and API secret")
		}
	}

	switch strings.TrimSpace(all[SettingHyperbeamRegion.Key]) {
	case "", "NA", "EU", "AS":
	default:
		return fmt.Errorf("shared browser region must be NA, EU or AS")
	}

	if err := validateChatbotSettings(all); err != nil {
		return err
	}

	emailProvider := EmailProvider(all[SettingEmailProvider.Key])
	if emailProvider != EmailProviderSMTP && emailProvider != EmailProviderCloudflare {
		return fmt.Errorf("email provider must be '%s' or '%s'", EmailProviderSMTP, EmailProviderCloudflare)
	}

	if emailProvider == EmailProviderCloudflare {
		if all[SettingCloudflareAccountID.Key] == "" || all[SettingCloudflareAPIToken.Key] == "" || all[SettingCloudflareEmailFrom.Key] == "" {
			return fmt.Errorf("cloudflare email requires account ID, API token and from address")
		}
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
	fcmCredentialsFile := os.Getenv("FCM_CREDENTIALS_FILE")

	Cfg = Config{
		Postgres:           pg,
		DatabaseURL:        databaseURL,
		GiphyAPIKey:        giphyKey,
		FCMCredentialsFile: fcmCredentialsFile,
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

func IsAppOrigin(origin string) bool {
	switch origin {
	case "capacitor://localhost", "ionic://localhost", "https://localhost", "http://localhost":
		return true
	default:
		return false
	}
}

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
