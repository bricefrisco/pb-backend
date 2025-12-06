package albion_bb

import (
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const KillsRetentionDays = 14

// CreateKillsSchema creates the kills collection if it doesn't exist
func CreateKillsSchema(app *pocketbase.PocketBase) error {
	return createKillsCollection(app)
}

func createKillsCollection(app *pocketbase.PocketBase) error {
	existing, _ := app.FindCollectionByNameOrId("kills")
	if existing != nil {
		return nil // Already exists
	}

	collection := core.NewBaseCollection("kills")

	// Event info
	collection.Fields.Add(&core.NumberField{
		Name:     "event_id",
		Required: true,
	})
	collection.Fields.Add(&core.DateField{
		Name:     "timestamp",
		Required: true,
	})

	// Killer info
	collection.Fields.Add(&core.TextField{
		Name:     "killer_name",
		Required: true,
	})
	collection.Fields.Add(&core.TextField{
		Name: "killer_guild",
	})
	collection.Fields.Add(&core.TextField{
		Name: "killer_alliance",
	})
	collection.Fields.Add(&core.TextField{
		Name: "killer_weapon",
	})
	collection.Fields.Add(&core.NumberField{
		Name: "killer_ip",
	})

	// Victim info
	collection.Fields.Add(&core.TextField{
		Name:     "victim_name",
		Required: true,
	})
	collection.Fields.Add(&core.TextField{
		Name: "victim_guild",
	})
	collection.Fields.Add(&core.TextField{
		Name: "victim_alliance",
	})
	collection.Fields.Add(&core.TextField{
		Name: "victim_weapon",
	})
	collection.Fields.Add(&core.NumberField{
		Name: "victim_ip",
	})

	// Participant count
	collection.Fields.Add(&core.NumberField{
		Name: "participant_count",
	})

	// Fame
	collection.Fields.Add(&core.NumberField{
		Name: "fame",
	})

	// Indexes for query patterns:
	// 1. Find top 50 latest kills - ORDER BY timestamp DESC
	// 2. Find top 50 latest kills where guild/alliance starts with 'ABC'
	// 3. Find top 50 latest kills where player name starts with 'ABC'
	collection.Indexes = []string{
		"CREATE UNIQUE INDEX idx_kills_event_id ON kills (event_id)",
		"CREATE INDEX idx_kills_timestamp ON kills (timestamp DESC)",
		"CREATE INDEX idx_kills_killer_name ON kills (killer_name COLLATE NOCASE)",
		"CREATE INDEX idx_kills_victim_name ON kills (victim_name COLLATE NOCASE)",
		"CREATE INDEX idx_kills_killer_guild ON kills (killer_guild COLLATE NOCASE)",
		"CREATE INDEX idx_kills_victim_guild ON kills (victim_guild COLLATE NOCASE)",
		"CREATE INDEX idx_kills_killer_alliance ON kills (killer_alliance COLLATE NOCASE)",
		"CREATE INDEX idx_kills_victim_alliance ON kills (victim_alliance COLLATE NOCASE)",
	}

	return app.Save(collection)
}

// SaveKills saves multiple kill records in a single transaction, skipping duplicates.
// existingIds is a set of event IDs that already exist in the database.
func SaveKills(app *pocketbase.PocketBase, kills []KillResponse, existingIds map[int]bool) (saved int, skipped int, errCount int) {
	if len(kills) == 0 {
		return 0, 0, 0
	}

	// Deduplicate kills array (same kill can appear on multiple pages due to offset shifting)
	seenIds := make(map[int]bool)
	uniqueKills := make([]KillResponse, 0)
	for _, kill := range kills {
		if !seenIds[kill.EventId] {
			seenIds[kill.EventId] = true
			uniqueKills = append(uniqueKills, kill)
		}
	}
	duplicatesInBatch := len(kills) - len(uniqueKills)

	// Filter out kills that already exist in database
	newKills := make([]KillResponse, 0)
	for _, kill := range uniqueKills {
		if existingIds[kill.EventId] {
			skipped++
		} else {
			newKills = append(newKills, kill)
		}
	}
	skipped += duplicatesInBatch

	if len(newKills) == 0 {
		return 0, skipped, 0
	}

	// Get collection once
	killsCollection, err := app.FindCollectionByNameOrId("kills")
	if err != nil {
		log.Printf("Failed to find kills collection: %v", err)
		return 0, skipped, len(newKills)
	}

	// Save all records in a single transaction
	err = app.RunInTransaction(func(txApp core.App) error {
		for _, kill := range newKills {
			record := core.NewRecord(killsCollection)
			populateKillRecord(record, kill)

			if err := txApp.Save(record); err != nil {
				return fmt.Errorf("failed to save kill %d: %w", kill.EventId, err)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("Transaction failed: %v", err)
		return 0, skipped, len(newKills)
	}

	return len(newKills), skipped, 0
}

// GetRecentEventIds fetches the most recent event IDs from the database in a single query.
// Returns a set of event IDs for fast lookup.
func GetRecentEventIds(app *pocketbase.PocketBase, limit int) map[int]bool {
	existingIds := make(map[int]bool)

	records, err := app.FindRecordsByFilter(
		"kills",
		"",           // no filter - get all
		"-timestamp", // sort by timestamp descending (most recent first)
		limit,
		0,
	)
	if err != nil {
		return existingIds
	}

	for _, record := range records {
		eventId := record.GetInt("event_id")
		existingIds[eventId] = true
	}

	return existingIds
}

func populateKillRecord(record *core.Record, kill KillResponse) {
	record.Set("event_id", kill.EventId)
	record.Set("timestamp", kill.TimeStamp)

	record.Set("killer_name", kill.Killer.Name)
	record.Set("killer_guild", kill.Killer.GuildName)
	record.Set("killer_alliance", kill.Killer.AllianceName)
	record.Set("killer_weapon", getWeaponType(kill.Killer.Equipment))
	record.Set("killer_ip", kill.Killer.AverageItemPower)

	record.Set("victim_name", kill.Victim.Name)
	record.Set("victim_guild", kill.Victim.GuildName)
	record.Set("victim_alliance", kill.Victim.AllianceName)
	record.Set("victim_weapon", getWeaponType(kill.Victim.Equipment))
	record.Set("victim_ip", kill.Victim.AverageItemPower)

	record.Set("participant_count", len(kill.Participants))
	record.Set("fame", kill.TotalVictimKillFame)
}

func getWeaponType(equipment KillEquipmentResponse) string {
	if equipment.MainHand != nil {
		return equipment.MainHand.Type
	}
	return ""
}

// CleanupOldKills deletes kills older than the retention period (14 days)
func CleanupOldKills(app *pocketbase.PocketBase) (deleted int, err error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -KillsRetentionDays)
	cutoffStr := cutoff.Format("2006-01-02 15:04:05.000Z")

	records, err := app.FindRecordsByFilter(
		"kills",
		"timestamp < {:cutoff}",
		"-timestamp",
		0, // no limit
		0,
		map[string]any{"cutoff": cutoffStr},
	)
	if err != nil {
		return 0, fmt.Errorf("failed to find old kills: %w", err)
	}

	if len(records) == 0 {
		return 0, nil
	}

	err = app.RunInTransaction(func(txApp core.App) error {
		for _, record := range records {
			if err := txApp.Delete(record); err != nil {
				return fmt.Errorf("failed to delete kill %v: %w", record.Id, err)
			}
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	log.Printf("Cleaned up %d kills older than %d days", len(records), KillsRetentionDays)
	return len(records), nil
}
