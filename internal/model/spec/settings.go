package spec

import (
	"umineko_city_of_books/internal/config"

	"github.com/google/uuid"
)

type (
	SettingsReconcile struct {
		Missing   map[config.SiteSettingKey]string
		Stale     []config.SiteSettingKey
		UpdatedBy uuid.UUID
	}

	SettingsUpdate struct {
		Key       config.SiteSettingKey
		Value     string
		UpdatedBy uuid.UUID
	}

	SettingsBulkUpdate struct {
		Values    map[config.SiteSettingKey]string
		UpdatedBy uuid.UUID
	}
)
