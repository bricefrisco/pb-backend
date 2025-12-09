package main

import (
	"log"
	"pb-backend/albion_bb"
	"pb-backend/chattanooga_homes"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	// Flags used for development; should all be true for production
	enableAlbion := false
	enableChattanoogaHomes := true

	app := pocketbase.New()

	// Register hooks for real-time change logging
	if enableChattanoogaHomes {
		chattanooga_homes.RegisterHooks(app)
	}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Albion kills
		if enableAlbion {
			if err := albion_bb.CreateKillsSchema(app); err != nil {
				log.Printf("Error creating kills schema: %v", err)
			}
			albionScheduler := albion_bb.NewScheduler(app)
			albionScheduler.Start()
		}

		// Chattanooga Homes
		if enableChattanoogaHomes {
			if err := chattanooga_homes.CreateHomesSchema(app); err != nil {
				log.Printf("Error creating homes schema: %v", err)
			}
			if err := chattanooga_homes.CreateDiscordConfigSchema(app); err != nil {
				log.Printf("Error creating discord config schema: %v", err)
			}
			homesScheduler := chattanooga_homes.NewHomesScheduler(app)
			homesScheduler.Start()
		}

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
