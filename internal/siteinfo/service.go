package siteinfo

import (
	"context"
	"maps"

	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/gameroom"
	"umineko_city_of_books/internal/mystery"
	"umineko_city_of_books/internal/secrets"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/usersecret"
	"umineko_city_of_books/internal/vanityrole"
)

type (
	Service interface {
		Get(ctx context.Context) dto.SiteInfoResponse
	}

	service struct {
		settingsSvc   settings.Service
		mysterySvc    mystery.Service
		gameRoomSvc   gameroom.Service
		vanityRoleSvc vanityrole.Service
		userSecretSvc usersecret.Service
		authSvc       auth.Service
	}
)

func NewService(
	settingsSvc settings.Service,
	mysterySvc mystery.Service,
	gameRoomSvc gameroom.Service,
	vanityRoleSvc vanityrole.Service,
	userSecretSvc usersecret.Service,
	authSvc auth.Service,
) Service {
	return &service{
		settingsSvc:   settingsSvc,
		mysterySvc:    mysterySvc,
		gameRoomSvc:   gameRoomSvc,
		vanityRoleSvc: vanityRoleSvc,
		userSecretSvc: userSecretSvc,
		authSvc:       authSvc,
	}
}

func (s *service) Get(ctx context.Context) dto.SiteInfoResponse {
	topDetectives, _ := s.mysterySvc.GetTopDetectiveIDs(ctx)
	topGMs, _ := s.mysterySvc.GetTopGMIDs(ctx)
	topChess, _ := s.gameRoomSvc.GetTopWinnerIDs(ctx, dto.GameTypeChess)
	topCheckers, _ := s.gameRoomSvc.GetTopWinnerIDs(ctx, dto.GameTypeCheckers)
	topOthello, _ := s.gameRoomSvc.GetTopWinnerIDs(ctx, dto.GameTypeOthello)
	topMinesweeper, _ := s.gameRoomSvc.GetTopWinnerIDs(ctx, dto.GameTypeMinesweeper)

	vanityRoles, _ := s.vanityRoleSvc.List(ctx)
	manualAssignments, _ := s.vanityRoleSvc.GetAllAssignments(ctx)

	assignments := make(map[string][]string)
	maps.Copy(assignments, manualAssignments)

	for _, uid := range topDetectives {
		assignments[uid] = append(assignments[uid], "system_top_detective")
	}
	for _, uid := range topGMs {
		assignments[uid] = append(assignments[uid], "system_top_gm")
	}
	for _, uid := range topChess {
		assignments[uid] = append(assignments[uid], "system_top_chess")
	}
	for _, uid := range topCheckers {
		assignments[uid] = append(assignments[uid], "system_top_checkers")
	}
	for _, uid := range topOthello {
		assignments[uid] = append(assignments[uid], "system_top_othello")
	}
	for _, uid := range topMinesweeper {
		assignments[uid] = append(assignments[uid], "system_top_minesweeper")
	}
	for _, spec := range secrets.WithVanityRole() {
		holders, _ := s.userSecretSvc.GetUserIDsWithSecret(ctx, string(spec.ID))
		for _, uid := range holders {
			assignments[uid.String()] = append(assignments[uid.String()], spec.VanityRoleID)
		}
	}

	vrList := make([]dto.SiteInfoVanityRole, len(vanityRoles))
	for i, vr := range vanityRoles {
		vrList[i] = dto.SiteInfoVanityRole{
			ID:        vr.ID,
			Label:     vr.Label,
			Color:     vr.Color,
			IsSystem:  vr.IsSystem,
			SortOrder: vr.SortOrder,
		}
	}

	listedSpecs := secrets.Listed()
	listedSecrets := make([]dto.SiteInfoSecret, len(listedSpecs))
	for i, spec := range listedSpecs {
		solved, _ := s.userSecretSvc.IsSolvedByAnyone(ctx, string(spec.ID))
		pieces := make([]dto.SiteInfoSecretPiece, len(spec.Pieces))
		for j, p := range spec.Pieces {
			pieces[j] = dto.SiteInfoSecretPiece{
				ID:     string(p.ID),
				Letter: p.Letter,
				Tile:   p.Tile,
			}
		}
		listedSecrets[i] = dto.SiteInfoSecret{
			ID:               string(spec.ID),
			Title:            spec.Title,
			Description:      spec.Description,
			VanityRoleID:     spec.VanityRoleID,
			Icon:             spec.Icon,
			Pointer:          spec.Pointer,
			SolvedMessage:    spec.SolvedMessage,
			ReadyPlaceholder: spec.ReadyPlaceholder,
			PendingHint:      spec.PendingHint,
			Solved:           solved,
			Pieces:           pieces,
		}
	}

	return dto.SiteInfoResponse{
		SiteName:              s.settingsSvc.Get(ctx, config.SettingSiteName),
		SiteDescription:       s.settingsSvc.Get(ctx, config.SettingSiteDescription),
		RegistrationType:      s.settingsSvc.Get(ctx, config.SettingRegistrationType),
		AnnouncementBanner:    s.settingsSvc.Get(ctx, config.SettingAnnouncementBanner),
		DefaultTheme:          s.settingsSvc.Get(ctx, config.SettingDefaultTheme),
		MaintenanceMode:       s.settingsSvc.GetBool(ctx, config.SettingMaintenanceMode),
		MaintenanceTitle:      s.settingsSvc.Get(ctx, config.SettingMaintenanceTitle),
		MaintenanceMessage:    s.settingsSvc.Get(ctx, config.SettingMaintenanceMessage),
		TurnstileEnabled:      s.settingsSvc.GetBool(ctx, config.SettingTurnstileEnabled),
		TurnstileSiteKey:      s.settingsSvc.Get(ctx, config.SettingTurnstileSiteKey),
		VoiceEnabled:          s.settingsSvc.GetBool(ctx, config.SettingVoiceEnabled),
		EmailEnabled:          s.authSvc.EmailEnabled(ctx),
		MaxImageSize:          s.settingsSvc.GetInt(ctx, config.SettingMaxImageSize),
		MaxVideoSize:          s.settingsSvc.GetInt(ctx, config.SettingMaxVideoSize),
		TopDetectiveIDs:       topDetectives,
		TopGMIDs:              topGMs,
		TopChessIDs:           topChess,
		TopCheckersIDs:        topCheckers,
		TopOthelloIDs:         topOthello,
		TopMinesweeperIDs:     topMinesweeper,
		VanityRoles:           vrList,
		VanityRoleAssignments: assignments,
		ListedSecrets:         listedSecrets,
		RulesPage:             s.settingsSvc.Get(ctx, config.SettingRulesPage),
		Version:               config.Version,
		AppLatestVersion:      s.settingsSvc.Get(ctx, config.SettingAppLatestVersion),
		AppDownloadURL:        s.settingsSvc.Get(ctx, config.SettingAppDownloadURL),
	}
}
