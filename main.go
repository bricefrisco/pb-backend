package main

import (
	"log"
	"pb-backend/albion_bb"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Create schema if it doesn't exist
		if err := albion_bb.CreateKillsSchema(app); err != nil {
			log.Printf("Error creating kills schema: %v", err)
		}

		api := albion_bb.NewAlbionAPI()

		// Fetch kills every 10 seconds
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				// Get recent event IDs from DB (single query)
				existingIds := albion_bb.GetRecentEventIds(app, 500)

				// Fetch kills, using existingIds to determine pagination
				kills, err := api.FetchRecentKillsUntilOverlap(51, existingIds)
				if err != nil {
					log.Printf("Error fetching recent kills: %v", err)
					// Continue anyway - we may have partial results
				}

				if len(kills) > 0 {
					// Save kills, reusing the same existingIds
					saved, skipped, errors := albion_bb.SaveKills(app, kills, existingIds)
					log.Printf("Kills: %d fetched, %d saved, %d skipped (duplicates), %d errors", len(kills), saved, skipped, errors)
				}
			}
		}()

		// Cleanup old kills every hour
		go func() {
			// Run cleanup immediately on startup
			albion_bb.CleanupOldKills(app)

			ticker := time.NewTicker(1 * time.Hour)
			defer ticker.Stop()

			for range ticker.C {
				albion_bb.CleanupOldKills(app)
			}
		}()

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
