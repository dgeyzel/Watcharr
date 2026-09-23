package feature

import (
	"github.com/sbondCo/Watcharr/config"
)

type ServerFeatures struct {
	Games bool `json:"games"`
}

type Service struct {
	cfg *config.ServerConfig
}

func NewService(cfg *config.ServerConfig) *Service {
	return &Service{
		cfg,
	}
}

// Get enabled server functionality from Config.
// Mainly so the frontend can store this once and know
// which btns should be shown, etc.
func (s *Service) GetEnabledFeatures() ServerFeatures {
	var f ServerFeatures
	if s.cfg.TwitchEnabled() {
		f.Games = true
	}
	return f
}
