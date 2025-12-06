package chattanooga_homes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Discord API endpoints
const (
	discordAPIBase = "https://discord.com/api/v10"
)

// DiscordConfig holds the Discord bot configuration
type DiscordConfig struct {
	BotToken       string
	HomesChannelID string
}

// DiscordEmbed represents a Discord embed message
type DiscordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
	Image       *DiscordEmbedImage  `json:"image,omitempty"`
	Thumbnail   *DiscordEmbedImage  `json:"thumbnail,omitempty"`
	URL         string              `json:"url,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
}

type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type DiscordEmbedImage struct {
	URL string `json:"url"`
}

type DiscordMessage struct {
	Content string         `json:"content,omitempty"`
	Embeds  []DiscordEmbed `json:"embeds,omitempty"`
}

type DiscordMessageResponse struct {
	ID string `json:"id"`
}

// CreateDiscordConfigSchema creates the discord_config collection
func CreateDiscordConfigSchema(app *pocketbase.PocketBase) error {
	existing, _ := app.FindCollectionByNameOrId("discord_config")
	if existing != nil {
		return nil
	}

	collection := core.NewBaseCollection("discord_config")

	collection.Fields.Add(&core.TextField{
		Name:     "name",
		Required: true,
	})
	collection.Fields.Add(&core.TextField{
		Name:     "bot_token",
		Required: true,
	})
	collection.Fields.Add(&core.TextField{
		Name:     "homes_channel_id",
		Required: true,
	})

	collection.Indexes = []string{
		"CREATE UNIQUE INDEX idx_discord_config_name ON discord_config (name)",
	}

	return app.Save(collection)
}

// GetDiscordConfig fetches the Discord configuration from the database
func GetDiscordConfig(app *pocketbase.PocketBase) (*DiscordConfig, error) {
	record, err := app.FindFirstRecordByFilter("discord_config", "name = 'default'")
	if err != nil {
		return nil, fmt.Errorf("discord config not found: %w", err)
	}

	return &DiscordConfig{
		BotToken:       record.GetString("bot_token"),
		HomesChannelID: record.GetString("homes_channel_id"),
	}, nil
}

// PostHomeToDiscord posts a new home listing to the Discord channel
func PostHomeToDiscord(app *pocketbase.PocketBase, record *core.Record) (string, error) {
	config, err := GetDiscordConfig(app)
	if err != nil {
		return "", err
	}

	embed := buildHomeEmbed(record, false)

	messageID, err := sendDiscordMessage(config, config.HomesChannelID, DiscordMessage{
		Embeds: []DiscordEmbed{embed},
	})
	if err != nil {
		return "", err
	}

	return messageID, nil
}

// PostUpdateToDiscordThread creates a thread and posts the update
func PostUpdateToDiscordThread(app *pocketbase.PocketBase, record *core.Record, changes []FieldChange) error {
	config, err := GetDiscordConfig(app)
	if err != nil {
		return err
	}

	messageID := record.GetString("discord_message_id")
	if messageID == "" {
		return fmt.Errorf("no discord message ID found for listing")
	}

	// Create or get thread from the original message
	threadID, err := createThreadFromMessage(config, config.HomesChannelID, messageID,
		fmt.Sprintf("Updates: %s", record.GetString("street")))
	if err != nil {
		return fmt.Errorf("failed to create thread: %w", err)
	}

	// Post update to the thread
	embed := buildUpdateEmbed(record, changes)
	_, err = sendDiscordMessage(config, threadID, DiscordMessage{
		Embeds: []DiscordEmbed{embed},
	})

	return err
}

// buildHomeEmbed creates a Discord embed for a home listing
func buildHomeEmbed(record *core.Record, isUpdate bool) DiscordEmbed {
	street := record.GetString("street")
	city := record.GetString("city")
	state := record.GetString("state")
	zip := record.GetString("zip")
	price := record.GetInt("price")
	beds := record.GetInt("beds_total")
	baths := record.GetFloat("baths_total")
	sqft := record.GetInt("living_area")
	acres := record.GetFloat("acres")
	yearBuilt := record.GetInt("year_built")
	subType := record.GetString("sub_type")
	county := record.GetString("county")
	url := record.GetString("url")
	imageURL := record.GetString("image_url")

	title := fmt.Sprintf("🏠 %s", street)
	if isUpdate {
		title = fmt.Sprintf("📝 Updated: %s", street)
	}

	// Green for new, blue for update
	color := 0x2ECC71 // Green
	if isUpdate {
		color = 0x3498DB // Blue
	}

	embed := DiscordEmbed{
		Title:     title,
		URL:       url,
		Color:     color,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Fields: []DiscordEmbedField{
			{Name: "💰 Price", Value: fmt.Sprintf("$%s", formatNumber(price)), Inline: true},
			{Name: "📍 Location", Value: fmt.Sprintf("%s, %s %s", city, state, zip), Inline: true},
			{Name: "🏘️ Type", Value: subType, Inline: true},
			{Name: "🛏️ Beds", Value: fmt.Sprintf("%d", beds), Inline: true},
			{Name: "🛁 Baths", Value: fmt.Sprintf("%.1f", baths), Inline: true},
			{Name: "📐 Sq Ft", Value: formatNumber(sqft), Inline: true},
			{Name: "🌳 Acres", Value: fmt.Sprintf("%.2f", acres), Inline: true},
			{Name: "📅 Year Built", Value: fmt.Sprintf("%d", yearBuilt), Inline: true},
			{Name: "🗺️ County", Value: county, Inline: true},
		},
	}

	if imageURL != "" {
		// Use full-size image to make it prominent (Discord shows near the top of the embed stack).
		embed.Image = &DiscordEmbedImage{URL: imageURL}
	}

	return embed
}

// buildUpdateEmbed creates a Discord embed for listing updates
func buildUpdateEmbed(record *core.Record, changes []FieldChange) DiscordEmbed {
	street := record.GetString("street")

	// Build fields for each change with old -> new format
	var fields []DiscordEmbedField

	for _, change := range changes {
		fieldName := formatFieldName(change.Field)
		oldStr := formatFieldValue(change.Field, change.OldValue)
		newStr := formatFieldValue(change.Field, change.NewValue)

		fields = append(fields, DiscordEmbedField{
			Name:   fieldName,
			Value:  fmt.Sprintf("%s → %s", oldStr, newStr),
			Inline: true,
		})
	}

	return DiscordEmbed{
		Title:     fmt.Sprintf("📝 %s", street),
		Color:     0xF39C12, // Orange
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Fields:    fields,
	}
}

// formatFieldName converts field names to display names with emojis
func formatFieldName(field string) string {
	names := map[string]string{
		"price":       "💰 Price",
		"status":      "📊 Status",
		"street":      "🏠 Street",
		"city":        "📍 City",
		"state":       "📍 State",
		"zip":         "📍 Zip",
		"sub_type":    "🏘️ Type",
		"county":      "🗺️ County",
		"area":        "🗺️ Area",
		"subdivision": "🏘️ Subdivision",
		"living_area": "📐 Sq Ft",
		"beds_total":  "🛏️ Beds",
		"baths_total": "🛁 Baths",
		"acres":       "🌳 Acres",
		"year_built":  "📅 Year Built",
		"url":         "🔗 URL",
		"image_url":   "🖼️ Image",
	}
	if name, ok := names[field]; ok {
		return name
	}
	return field
}

// formatFieldValue formats a field value for display
func formatFieldValue(field string, value interface{}) string {
	if value == nil {
		return "N/A"
	}

	switch field {
	case "price":
		if v, ok := value.(float64); ok {
			return fmt.Sprintf("$%s", formatNumber(int(v)))
		}
		if v, ok := value.(int); ok {
			return fmt.Sprintf("$%s", formatNumber(v))
		}
	case "living_area":
		if v, ok := value.(float64); ok {
			return fmt.Sprintf("%s sq ft", formatNumber(int(v)))
		}
		if v, ok := value.(int); ok {
			return fmt.Sprintf("%s sq ft", formatNumber(v))
		}
	case "acres":
		if v, ok := value.(float64); ok {
			return fmt.Sprintf("%.2f", v)
		}
	case "baths_total":
		if v, ok := value.(float64); ok {
			return fmt.Sprintf("%.1f", v)
		}
	}

	return fmt.Sprintf("%v", value)
}

// sendDiscordMessage sends a message to a Discord channel
func sendDiscordMessage(config *DiscordConfig, channelID string, message DiscordMessage) (string, error) {
	url := fmt.Sprintf("%s/channels/%s/messages", discordAPIBase, channelID)

	body, err := json.Marshal(message)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bot "+config.BotToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("discord API error: %s - %s", resp.Status, string(respBody))
	}

	var msgResp DiscordMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&msgResp); err != nil {
		return "", err
	}

	return msgResp.ID, nil
}

// createThreadFromMessage creates a thread from an existing message
func createThreadFromMessage(config *DiscordConfig, channelID, messageID, threadName string) (string, error) {
	url := fmt.Sprintf("%s/channels/%s/messages/%s/threads", discordAPIBase, channelID, messageID)

	body, err := json.Marshal(map[string]interface{}{
		"name":                  threadName,
		"auto_archive_duration": 1440, // 24 hours
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bot "+config.BotToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Thread already exists returns 400, try to find it
	if resp.StatusCode == http.StatusBadRequest {
		// Return the message ID as thread ID (Discord uses message ID as thread ID)
		return messageID, nil
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("discord API error creating thread: %s - %s", resp.Status, string(respBody))
	}

	var threadResp struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&threadResp); err != nil {
		return "", err
	}

	return threadResp.ID, nil
}

// formatNumber adds commas to numbers
func formatNumber(n int) string {
	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}

	var result []byte
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
