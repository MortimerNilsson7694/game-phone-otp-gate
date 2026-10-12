package game

import "testing"

func TestAssetModerationDecision(t *testing.T) {
	tests := []struct {
		name      string
		risk      int
		wantState string
		wantQueue int
	}{
		{name: "ordinary player banner", risk: 24, wantState: "accepted", wantQueue: 0},
		{name: "high-risk player banner", risk: 70, wantState: "pending_moderation", wantQueue: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore()
			state, err := store.AddAsset(Asset{ID: "asset-1", PlayerID: "player-7", Kind: "banner", RiskScore: tt.risk})
			if err != nil {
				t.Fatal(err)
			}
			if state != tt.wantState {
				t.Fatalf("state = %q, want %q", state, tt.wantState)
			}
			if got := len(store.ModerationQueue()); got != tt.wantQueue {
				t.Fatalf("queue length = %d, want %d", got, tt.wantQueue)
			}
		})
	}
}
