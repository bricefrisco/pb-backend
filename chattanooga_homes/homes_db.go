package chattanooga_homes

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Home represents a real estate listing
type Home struct {
	ListingID   string
	Street      string
	City        string
	State       string
	Zip         string
	Price       int
	SubType     string
	County      string
	Area        string
	Subdivision string
	LivingArea  int // square feet
	BedsTotal   int
	BathsTotal  float64
	Acres       float64
	YearBuilt   int
	URL         string
	ImageURL    string
	Status      string
}

// CreateHomesSchema creates the homes collection
func CreateHomesSchema(app *pocketbase.PocketBase) error {
	return createHomesCollection(app)
}

func createHomesCollection(app *pocketbase.PocketBase) error {
	existing, _ := app.FindCollectionByNameOrId("homes")
	if existing != nil {
		return nil
	}

	collection := core.NewBaseCollection("homes")

	// Listing ID (unique identifier from MLS)
	collection.Fields.Add(&core.TextField{
		Name:     "listing_id",
		Required: true,
	})

	// Address fields
	collection.Fields.Add(&core.TextField{
		Name:     "street",
		Required: true,
	})
	collection.Fields.Add(&core.TextField{
		Name: "city",
	})
	collection.Fields.Add(&core.TextField{
		Name: "state",
	})
	collection.Fields.Add(&core.TextField{
		Name: "zip",
	})

	// Current price
	collection.Fields.Add(&core.NumberField{
		Name: "price",
	})

	// Property details
	collection.Fields.Add(&core.TextField{
		Name: "sub_type",
	})
	collection.Fields.Add(&core.TextField{
		Name: "county",
	})
	collection.Fields.Add(&core.TextField{
		Name: "area",
	})
	collection.Fields.Add(&core.TextField{
		Name: "subdivision",
	})
	collection.Fields.Add(&core.NumberField{
		Name: "living_area",
	})
	collection.Fields.Add(&core.NumberField{
		Name: "beds_total",
	})
	collection.Fields.Add(&core.NumberField{
		Name: "baths_total",
	})
	collection.Fields.Add(&core.NumberField{
		Name: "acres",
	})
	collection.Fields.Add(&core.NumberField{
		Name: "year_built",
	})

	// URL to listing
	collection.Fields.Add(&core.TextField{
		Name: "url",
	})

	// Image URL
	collection.Fields.Add(&core.TextField{
		Name: "image_url",
	})

	// Tracking fields
	collection.Fields.Add(&core.DateField{
		Name: "first_seen",
	})
	collection.Fields.Add(&core.DateField{
		Name: "last_seen",
	})

	// Status: Active, Inactive, Sold, etc.
	collection.Fields.Add(&core.TextField{
		Name: "status",
	})

	// Discord message ID for thread creation
	collection.Fields.Add(&core.TextField{
		Name: "discord_message_id",
	})

	collection.Indexes = []string{
		"CREATE UNIQUE INDEX idx_homes_listing_id ON homes (listing_id)",
		"CREATE INDEX idx_homes_price ON homes (price)",
		"CREATE INDEX idx_homes_city ON homes (city)",
		"CREATE INDEX idx_homes_county ON homes (county)",
		"CREATE INDEX idx_homes_status ON homes (status)",
	}

	return app.Save(collection)
}

// SaveHomes saves or updates multiple home listings in a single transaction
func SaveHomes(app *pocketbase.PocketBase, homes []Home) (saved int, err error) {
	if len(homes) == 0 {
		return 0, nil
	}

	collection, err := app.FindCollectionByNameOrId("homes")
	if err != nil {
		return 0, fmt.Errorf("failed to find homes collection: %w", err)
	}

	now := time.Now().UTC()

	err = app.RunInTransaction(func(txApp core.App) error {
		for _, home := range homes {
			// Find existing or create new record
			filter := fmt.Sprintf("listing_id = '%s'", home.ListingID)
			record, _ := txApp.FindFirstRecordByFilter("homes", filter)

			if record == nil {
				record = core.NewRecord(collection)
				record.Set("listing_id", home.ListingID)
				record.Set("first_seen", now)
			}

			// Set all fields
			record.Set("street", home.Street)
			record.Set("city", home.City)
			record.Set("state", home.State)
			record.Set("zip", home.Zip)
			record.Set("price", home.Price)
			record.Set("sub_type", home.SubType)
			record.Set("county", home.County)
			record.Set("area", home.Area)
			record.Set("subdivision", home.Subdivision)
			record.Set("living_area", home.LivingArea)
			record.Set("beds_total", home.BedsTotal)
			record.Set("baths_total", home.BathsTotal)
			record.Set("acres", home.Acres)
			record.Set("year_built", home.YearBuilt)
			record.Set("url", home.URL)
			record.Set("image_url", home.ImageURL)
			record.Set("last_seen", now)
			record.Set("status", home.Status)

			if err := txApp.Save(record); err != nil {
				return fmt.Errorf("failed to save home %s: %w", home.ListingID, err)
			}
			saved++
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	return saved, nil
}
