package chattanooga_homes

import (
	"log"
	"time"

	"github.com/pocketbase/pocketbase"
)

const (
	// Scrape every minute for real-time updates
	scrapeInterval = 1 * time.Minute
)

// HomesScheduler handles periodic scraping of home listings
type HomesScheduler struct {
	app     *pocketbase.PocketBase
	scraper *Scraper
}

// NewHomesScheduler creates a new scheduler instance
func NewHomesScheduler(app *pocketbase.PocketBase) *HomesScheduler {
	return &HomesScheduler{
		app:     app,
		scraper: NewScraper(),
	}
}

// Start begins the scheduler goroutine for scraping
func (s *HomesScheduler) Start() {
	go s.runScrapeLoop()
}

func (s *HomesScheduler) runScrapeLoop() {
	ticker := time.NewTicker(scrapeInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.scrapeAndSaveHomes()
	}
}

func (s *HomesScheduler) scrapeAndSaveHomes() {
	log.Println("Starting home listings scrape...")

	homes, err := s.scraper.ScrapeListings()
	if err != nil {
		log.Printf("Error scraping listings: %v", err)
		return
	}

	if len(homes) == 0 {
		log.Println("No homes found in scrape")
		return
	}

	log.Printf("Found %d listings, saving to database...", len(homes))

	saved, err := SaveHomes(s.app, homes)
	if err != nil {
		log.Printf("Error saving homes: %v", err)
	}

	log.Printf("Scrape complete: %d saved", saved)
}

// ScrapeNow triggers an immediate scrape (useful for Discord commands)
func (s *HomesScheduler) ScrapeNow() (int, error) {
	homes, err := s.scraper.ScrapeListings()
	if err != nil {
		return 0, err
	}

	return SaveHomes(s.app, homes)
}
