package albion_bb

import (
	"log"
	"time"

	"github.com/pocketbase/pocketbase"
)

const (
	fetchInterval   = 10 * time.Second
	cleanupInterval = 1 * time.Hour
	pageSize        = 51
	recentIdsLimit  = 500
)

// Scheduler handles periodic fetching and cleanup of kills.
type Scheduler struct {
	app *pocketbase.PocketBase
	api *AlbionAPI
}

// NewScheduler creates a new scheduler instance.
func NewScheduler(app *pocketbase.PocketBase) *Scheduler {
	return &Scheduler{
		app: app,
		api: NewAlbionAPI(),
	}
}

// Start begins the scheduler goroutines for fetching and cleanup.
func (s *Scheduler) Start() {
	go s.runFetchLoop()
	go s.runCleanupLoop()
}

func (s *Scheduler) runFetchLoop() {
	ticker := time.NewTicker(fetchInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.fetchAndSaveKills()
	}
}

func (s *Scheduler) fetchAndSaveKills() {
	// Get recent event IDs from DB (single query)
	existingIds := GetRecentEventIds(s.app, recentIdsLimit)

	// Fetch kills, using existingIds to determine pagination
	kills, err := s.api.FetchRecentKillsUntilOverlap(pageSize, existingIds)
	if err != nil {
		log.Printf("Error fetching recent kills: %v", err)
		// Continue anyway - we may have partial results
	}

	if len(kills) > 0 {
		// Save kills, reusing the same existingIds
		saved, skipped, errors := SaveKills(s.app, kills, existingIds)
		log.Printf("Kills: %d fetched, %d saved, %d skipped (duplicates), %d errors", len(kills), saved, skipped, errors)
	}
}

func (s *Scheduler) runCleanupLoop() {
	// Run cleanup immediately on startup
	CleanupOldKills(s.app)

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		CleanupOldKills(s.app)
	}
}
