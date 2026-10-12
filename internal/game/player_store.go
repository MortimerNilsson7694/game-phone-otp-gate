package game

import (
	"errors"
	"sync"
	"time"
)

type Asset struct {
	ID        string `json:"id"`
	PlayerID  string `json:"player_id"`
	Kind      string `json:"kind"`
	RiskScore int    `json:"risk_score"`
}

type LiveEvent struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	StartsAt  time.Time `json:"starts_at"`
	CreatedBy string    `json:"created_by"`
}

type ModerationItem struct {
	AssetID string `json:"asset_id"`
	Reason  string `json:"reason"`
	State   string `json:"state"`
}

type Store struct {
	mu         sync.Mutex
	assets     map[string]Asset
	events     map[string]LiveEvent
	moderation []ModerationItem
}

func NewStore() *Store {
	return &Store{assets: make(map[string]Asset), events: make(map[string]LiveEvent)}
}

func (s *Store) AddAsset(asset Asset) (string, error) {
	if asset.ID == "" || asset.PlayerID == "" || asset.Kind == "" {
		return "", errors.New("id, player_id, and kind are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.assets[asset.ID]; exists {
		return "", errors.New("asset id already exists")
	}
	s.assets[asset.ID] = asset
	if asset.RiskScore >= 70 {
		s.moderation = append(s.moderation, ModerationItem{AssetID: asset.ID, Reason: "risk_score", State: "pending"})
		return "pending_moderation", nil
	}
	return "accepted", nil
}

func (s *Store) AddEvent(event LiveEvent) error {
	if event.ID == "" || event.Name == "" || event.CreatedBy == "" || event.StartsAt.IsZero() {
		return errors.New("id, name, starts_at, and created_by are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.events[event.ID]; exists {
		return errors.New("event id already exists")
	}
	s.events[event.ID] = event
	return nil
}

func (s *Store) ModerationQueue() []ModerationItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]ModerationItem(nil), s.moderation...)
}
