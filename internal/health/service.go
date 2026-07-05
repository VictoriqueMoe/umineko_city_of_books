package health

import (
	"context"
	"database/sql"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/livekit"
	settingssvc "umineko_city_of_books/internal/settings"

	healthgo "github.com/hellofresh/health-go/v5"
)

type (
	Service interface {
		Measure(ctx context.Context) healthgo.Check
	}

	service struct {
		health *healthgo.Health
	}
)

func NewService(db *sql.DB, version string, settingsService settingssvc.Service, livekitService livekit.Service) (Service, error) {
	siteName := settingsService.Get(context.Background(), config.SettingSiteName)
	checker, err := healthgo.New(
		healthgo.WithComponent(healthgo.Component{
			Name:    siteName,
			Version: version,
		}),
		healthgo.WithChecks(
			healthgo.Config{
				Name:      "postgres",
				Timeout:   3 * time.Second,
				SkipOnErr: false,
				Check: func(ctx context.Context) error {
					return db.PingContext(ctx)
				},
			},
			healthgo.Config{
				Name:      "livekit",
				Timeout:   3 * time.Second,
				SkipOnErr: true,
				Check: func(ctx context.Context) error {
					if !livekitService.Enabled() {
						return nil
					}

					_, err := livekitService.ActiveRooms(ctx)
					return err
				},
			},
		),
	)
	if err != nil {
		return nil, err
	}

	return &service{health: checker}, nil
}

func (s *service) Measure(ctx context.Context) healthgo.Check {
	return s.health.Measure(ctx)
}
