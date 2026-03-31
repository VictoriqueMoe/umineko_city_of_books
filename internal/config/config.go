package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type (
	Config struct {
		DBPath string
	}

	SettingType int

	SiteSettingKey string

	SiteSettingDef struct {
		Key     SiteSettingKey
		Default string
		Type    SettingType
	}
)

const (
	TypeString SettingType = iota
	TypeBool
	TypeInt
)

var (
	Cfg Config

	SettingUploadDir           = SiteSettingDef{"upload_dir", "uploads", TypeString}
	SettingBaseURL             = SiteSettingDef{"base_url", "http://localhost:4323", TypeString}
	SettingLogLevel            = SiteSettingDef{"log_level", "info", TypeString}
	SettingMaxBodySize         = SiteSettingDef{"max_body_size", "52428800", TypeInt}
	SettingMaxImageSize        = SiteSettingDef{"max_image_size", "10485760", TypeInt}
	SettingMaxVideoSize        = SiteSettingDef{"max_video_size", "104857600", TypeInt}
	SettingMaxGeneralSize      = SiteSettingDef{"max_general_size", "52428800", TypeInt}
	SettingRegistrationType    = SiteSettingDef{"registration_type", "open", TypeString}
	SettingMaintenanceMode     = SiteSettingDef{"maintenance_mode", "false", TypeBool}
	SettingSiteName            = SiteSettingDef{"site_name", "Umineko City of Books", TypeString}
	SettingSiteDescription     = SiteSettingDef{"site_description", "", TypeString}
	SettingAnnouncementBanner  = SiteSettingDef{"announcement_banner", "", TypeString}
	SettingMaxTheoriesPerDay   = SiteSettingDef{"max_theories_per_day", "0", TypeInt}
	SettingMaxResponsesPerDay  = SiteSettingDef{"max_responses_per_day", "0", TypeInt}
	SettingMinPasswordLength   = SiteSettingDef{"min_password_length", "8", TypeInt}
	SettingSessionDurationDays = SiteSettingDef{"session_duration_days", "30", TypeInt}
	SettingDefaultTheme        = SiteSettingDef{"default_theme", "featherine", TypeString}
	SettingDMsEnabled          = SiteSettingDef{"dms_enabled", "true", TypeBool}
	SettingTurnstileEnabled    = SiteSettingDef{"turnstile_enabled", "false", TypeBool}
	SettingTurnstileSiteKey    = SiteSettingDef{"turnstile_site_key", "", TypeString}
	SettingTurnstileSecretKey  = SiteSettingDef{"turnstile_secret_key", "", TypeString}

	AllSiteSettings = []SiteSettingDef{
		SettingUploadDir,
		SettingBaseURL,
		SettingLogLevel,
		SettingMaxBodySize,
		SettingMaxImageSize,
		SettingMaxVideoSize,
		SettingMaxGeneralSize,
		SettingRegistrationType,
		SettingMaintenanceMode,
		SettingSiteName,
		SettingSiteDescription,
		SettingAnnouncementBanner,
		SettingMaxTheoriesPerDay,
		SettingMaxResponsesPerDay,
		SettingMinPasswordLength,
		SettingSessionDurationDays,
		SettingDefaultTheme,
		SettingDMsEnabled,
		SettingTurnstileEnabled,
		SettingTurnstileSiteKey,
		SettingTurnstileSecretKey,
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
	_ = godotenv.Load()

	dbPath := "truths.db"
	if v, ok := os.LookupEnv("DB_PATH"); ok {
		dbPath = v
	}
	Cfg = Config{DBPath: dbPath}

	for i := range AllSiteSettings {
		envKey := strings.ToUpper(string(AllSiteSettings[i].Key))
		if v, ok := os.LookupEnv(envKey); ok {
			AllSiteSettings[i].Default = v
		}
	}
}
