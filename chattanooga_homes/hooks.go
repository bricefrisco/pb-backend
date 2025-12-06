package chattanooga_homes

import (
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// FieldChange represents a change to a field with old and new values
type FieldChange struct {
	Field    string
	OldValue interface{}
	NewValue interface{}
}

// RegisterHooks sets up PocketBase hooks for the homes collection
// These hooks will:
// 1. Log changes server-side
// 2. Post to Discord on create/update
// 3. Automatically broadcast to WebSocket subscribers (built into PocketBase)
func RegisterHooks(app *pocketbase.PocketBase) {
	// Hook: After a home record is created
	app.OnRecordAfterCreateSuccess("homes").BindFunc(func(e *core.RecordEvent) error {
		street := e.Record.GetString("street")
		price := e.Record.GetInt("price")
		city := e.Record.GetString("city")

		log.Printf("[HOMES EVENT] NEW LISTING: %s, %s - $%d", street, city, price)

		// Post to Discord and save message ID
		go func() {
			messageID, err := PostHomeToDiscord(app, e.Record)
			if err != nil {
				log.Printf("[DISCORD] Error posting new listing: %v", err)
				return
			}

			// Save the Discord message ID to the record
			e.Record.Set("discord_message_id", messageID)
			if err := app.Save(e.Record); err != nil {
				log.Printf("[DISCORD] Error saving message ID: %v", err)
			} else {
				log.Printf("[DISCORD] Posted new listing, message ID: %s", messageID)
			}
		}()

		return e.Next()
	})

	// Hook: After a home record is updated
	app.OnRecordAfterUpdateSuccess("homes").BindFunc(func(e *core.RecordEvent) error {
		// Get the original record before the update
		original := e.Record.Original()
		if original == nil {
			return e.Next()
		}

		// Check if any meaningful fields changed (exclude last_seen and discord_message_id)
		fieldsToCheck := []string{
			"street", "city", "state", "zip", "price", "sub_type", "county",
			"area", "subdivision", "living_area", "beds_total", "baths_total",
			"acres", "year_built", "url", "image_url", "status",
		}

		var changes []FieldChange
		for _, field := range fieldsToCheck {
			oldVal := original.Get(field)
			newVal := e.Record.Get(field)
			if oldVal != newVal {
				changes = append(changes, FieldChange{
					Field:    field,
					OldValue: oldVal,
					NewValue: newVal,
				})
			}
		}

		// Skip if no meaningful changes
		if len(changes) == 0 {
			return e.Next()
		}

		// Skip if this is a new record (original had no street - key required field)
		// This happens when we save discord_message_id right after creation
		if original.GetString("street") == "" {
			return e.Next()
		}

		street := e.Record.GetString("street")
		status := e.Record.GetString("status")
		price := e.Record.GetInt("price")

		log.Printf("[HOMES EVENT] UPDATED: %s - Status: %s, Price: $%d", street, status, price)
		for _, change := range changes {
			log.Printf("  - %s: %v -> %v", change.Field, change.OldValue, change.NewValue)
		}

		// Post update to Discord thread
		go func() {
			if err := PostUpdateToDiscordThread(app, e.Record, changes); err != nil {
				log.Printf("[DISCORD] Error posting update: %v", err)
			} else {
				log.Printf("[DISCORD] Posted update to thread for: %s", street)
			}
		}()

		return e.Next()
	})

	// Hook: After a home record is deleted
	app.OnRecordAfterDeleteSuccess("homes").BindFunc(func(e *core.RecordEvent) error {
		street := e.Record.GetString("street")

		log.Printf("[HOMES EVENT] DELETED: %s", street)

		return e.Next()
	})
}
