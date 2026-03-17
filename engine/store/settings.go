package store

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

var settingsFile = "settings.json"

// TeleportProfile holds reusable Teleport connection config (cluster, user, profile).
type TeleportProfile struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Cluster string `json:"cluster"`
	User    string `json:"user"`
	Profile string `json:"profile"`
}

// Settings holds app-wide settings including layout and Teleport profiles.
type Settings struct {
	LayoutDirection  string           `json:"layout_direction,omitempty"`
	TeleportProfiles []TeleportProfile `json:"teleport_profiles,omitempty"`
}

// LoadSettings loads settings from settings.json. Returns defaults if file missing.
func LoadSettings() (Settings, error) {
	fileMu.Lock()
	defer fileMu.Unlock()

	s := Settings{
		LayoutDirection:  "vertical",
		TeleportProfiles: []TeleportProfile{},
	}

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, err
	}

	if err := json.Unmarshal(data, &s); err != nil {
		return s, err
	}
	if s.TeleportProfiles == nil {
		s.TeleportProfiles = []TeleportProfile{}
	}
	if s.LayoutDirection == "" {
		s.LayoutDirection = "vertical"
	}
	return s, nil
}

// SaveSettings writes settings to settings.json.
func SaveSettings(s Settings) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	if dir := filepath.Dir(settingsFile); dir != "." {
		os.MkdirAll(dir, 0755)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsFile, data, 0644)
}

// GetTeleportProfile returns a Teleport profile by ID.
func GetTeleportProfile(id string) (*TeleportProfile, error) {
	s, err := LoadSettings()
	if err != nil {
		return nil, err
	}
	for i := range s.TeleportProfiles {
		if s.TeleportProfiles[i].ID == id {
			return &s.TeleportProfiles[i], nil
		}
	}
	return nil, nil
}

// ListTeleportProfiles returns all saved Teleport profiles.
func ListTeleportProfiles() ([]TeleportProfile, error) {
	s, err := LoadSettings()
	if err != nil {
		return nil, err
	}
	return s.TeleportProfiles, nil
}

// CreateTeleportProfile adds a new profile and saves.
func CreateTeleportProfile(p TeleportProfile) (TeleportProfile, error) {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	s, err := LoadSettings()
	if err != nil {
		return p, err
	}
	s.TeleportProfiles = append(s.TeleportProfiles, p)
	return p, SaveSettings(s)
}

// UpdateTeleportProfile updates an existing profile by ID.
func UpdateTeleportProfile(id string, p TeleportProfile) error {
	s, err := LoadSettings()
	if err != nil {
		return err
	}
	for i := range s.TeleportProfiles {
		if s.TeleportProfiles[i].ID == id {
			p.ID = id
			s.TeleportProfiles[i] = p
			return SaveSettings(s)
		}
	}
	return nil
}

// ResolveTeleportConfig fills in cluster/user/profile from a saved profile if TeleportProfileID is set.
// Returns the config unchanged if no profile ID or profile not found.
func ResolveTeleportConfig(cfg *ConnectionConfig) {
	if !cfg.TeleportEnabled || cfg.TeleportProfileID == "" {
		return
	}
	p, err := GetTeleportProfile(cfg.TeleportProfileID)
	if err != nil || p == nil {
		return
	}
	cfg.TeleportCluster = p.Cluster
	cfg.TeleportUser = p.User
	cfg.TeleportProfile = p.Profile
}

// DeleteTeleportProfile removes a profile by ID.
func DeleteTeleportProfile(id string) error {
	s, err := LoadSettings()
	if err != nil {
		return err
	}
	for i := range s.TeleportProfiles {
		if s.TeleportProfiles[i].ID == id {
			s.TeleportProfiles = append(s.TeleportProfiles[:i], s.TeleportProfiles[i+1:]...)
			return SaveSettings(s)
		}
	}
	return nil
}
