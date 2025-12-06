package albion_bb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type BattleAllianceResponse struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Kills    int    `json:"kills"`
	KillFame int    `json:"killFame"`
	Deaths   int    `json:"deaths"`
}

type BattleGuildResponse struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	AllianceId   string `json:"allianceId"`
	AllianceName string `json:"alliance"`
	KillFame     int    `json:"killFame"`
	Kills        int    `json:"kills"`
	Deaths       int    `json:"deaths"`
}

type BattlePlayerResponse struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	AllianceId   string `json:"allianceId"`
	AllianceName string `json:"allianceName"`
	GuildId      string `json:"guildId"`
	GuildName    string `json:"guildName"`
	KillFame     int    `json:"killFame"`
	Kills        int    `json:"kills"`
	Deaths       int    `json:"deaths"`
}

type BattleResponse struct {
	Id            int                               `json:"id"`
	BattleTimeout int                               `json:"battle_TIMEOUT"`
	StartTime     time.Time                         `json:"startTime"`
	EndTime       time.Time                         `json:"endTime"`
	TotalFame     int                               `json:"totalFame"`
	TotalKills    int                               `json:"totalKills"`
	Alliances     map[string]BattleAllianceResponse `json:"alliances"`
	Guilds        map[string]BattleGuildResponse    `json:"guilds"`
	Players       map[string]BattlePlayerResponse   `json:"players"`
}

type BattleKillItemResponse struct {
	Type    string `json:"Type"`
	Quality int    `json:"Quality"`
}

type BattleKillEquipmentResponse struct {
	MainHand BattleKillItemResponse `json:"MainHand"`
}

type BattleKillPlayerResponse struct {
	Id                 string                      `json:"Id"`
	Name               string                      `json:"Name"`
	AllianceId         string                      `json:"AllianceId"`
	AllianceName       string                      `json:"AllianceName"`
	GuildId            string                      `json:"GuildId"`
	GuildName          string                      `json:"GuildName"`
	KillFame           int                         `json:"KillFame"`
	DeathFame          int                         `json:"DeathFame"`
	AverageItemPower   float64                     `json:"AverageItemPower"`
	DamageDone         float64                     `json:"DamageDone"`
	SupportHealingDone float64                     `json:"SupportHealingDone"`
	Equipment          BattleKillEquipmentResponse `json:"Equipment"`
}

type BattleKillResponse struct {
	BattleId            int                        `json:"BattleId"`
	Timestamp           time.Time                  `json:"Timestamp"`
	Killer              BattleKillPlayerResponse   `json:"Killer"`
	Victim              BattleKillPlayerResponse   `json:"Victim"`
	TotalVictimKillFame int                        `json:"TotalVictimKillFame"`
	GroupMembers        []BattleKillPlayerResponse `json:"GroupMembers"`
	Participants        []BattleKillPlayerResponse `json:"Participants"`
}

// Events API models

type KillItemResponse struct {
	Type          string      `json:"Type"`
	Count         int         `json:"Count"`
	Quality       int         `json:"Quality"`
	ActiveSpells  []string    `json:"ActiveSpells"`
	PassiveSpells []string    `json:"PassiveSpells"`
	LegendarySoul interface{} `json:"LegendarySoul"`
}

type KillEquipmentResponse struct {
	MainHand *KillItemResponse `json:"MainHand"`
	OffHand  *KillItemResponse `json:"OffHand"`
	Head     *KillItemResponse `json:"Head"`
	Armor    *KillItemResponse `json:"Armor"`
	Shoes    *KillItemResponse `json:"Shoes"`
	Bag      *KillItemResponse `json:"Bag"`
	Cape     *KillItemResponse `json:"Cape"`
	Mount    *KillItemResponse `json:"Mount"`
	Potion   *KillItemResponse `json:"Potion"`
	Food     *KillItemResponse `json:"Food"`
}

type KillGatheringStatResponse struct {
	Total    int `json:"Total"`
	Royal    int `json:"Royal"`
	Outlands int `json:"Outlands"`
	Avalon   int `json:"Avalon"`
}

type KillGatheringStatsResponse struct {
	Fiber KillGatheringStatResponse `json:"Fiber"`
	Hide  KillGatheringStatResponse `json:"Hide"`
	Ore   KillGatheringStatResponse `json:"Ore"`
	Rock  KillGatheringStatResponse `json:"Rock"`
	Wood  KillGatheringStatResponse `json:"Wood"`
	All   KillGatheringStatResponse `json:"All"`
}

type KillPvEStatsResponse struct {
	Total            int `json:"Total"`
	Royal            int `json:"Royal"`
	Outlands         int `json:"Outlands"`
	Avalon           int `json:"Avalon"`
	Hellgate         int `json:"Hellgate"`
	CorruptedDungeon int `json:"CorruptedDungeon"`
	Mists            int `json:"Mists"`
}

type KillCraftingStatsResponse struct {
	Total    int `json:"Total"`
	Royal    int `json:"Royal"`
	Outlands int `json:"Outlands"`
	Avalon   int `json:"Avalon"`
}

type KillLifetimeStatisticsResponse struct {
	PvE           KillPvEStatsResponse       `json:"PvE"`
	Gathering     KillGatheringStatsResponse `json:"Gathering"`
	Crafting      KillCraftingStatsResponse  `json:"Crafting"`
	CrystalLeague int                        `json:"CrystalLeague"`
	FishingFame   int                        `json:"FishingFame"`
	FarmingFame   int                        `json:"FarmingFame"`
	Timestamp     *time.Time                 `json:"Timestamp"`
}

type KillPlayerResponse struct {
	Id                 string                         `json:"Id"`
	Name               string                         `json:"Name"`
	GuildId            string                         `json:"GuildId"`
	GuildName          string                         `json:"GuildName"`
	AllianceId         string                         `json:"AllianceId"`
	AllianceName       string                         `json:"AllianceName"`
	AllianceTag        string                         `json:"AllianceTag"`
	Avatar             string                         `json:"Avatar"`
	AvatarRing         string                         `json:"AvatarRing"`
	KillFame           int                            `json:"KillFame"`
	DeathFame          int                            `json:"DeathFame"`
	FameRatio          float64                        `json:"FameRatio"`
	AverageItemPower   float64                        `json:"AverageItemPower"`
	DamageDone         float64                        `json:"DamageDone"`
	SupportHealingDone float64                        `json:"SupportHealingDone"`
	Equipment          KillEquipmentResponse          `json:"Equipment"`
	Inventory          []*KillItemResponse            `json:"Inventory"`
	LifetimeStatistics KillLifetimeStatisticsResponse `json:"LifetimeStatistics"`
}

type KillResponse struct {
	EventId              int                  `json:"EventId"`
	TimeStamp            time.Time            `json:"TimeStamp"`
	Version              int                  `json:"Version"`
	BattleId             int                  `json:"BattleId"`
	KillArea             string               `json:"KillArea"`
	Category             *string              `json:"Category"`
	Type                 string               `json:"Type"`
	GroupMemberCount     int                  `json:"groupMemberCount"`
	NumberOfParticipants int                  `json:"numberOfParticipants"`
	Killer               KillPlayerResponse   `json:"Killer"`
	Victim               KillPlayerResponse   `json:"Victim"`
	TotalVictimKillFame  int                  `json:"TotalVictimKillFame"`
	Location             *string              `json:"Location"`
	GvGMatch             *interface{}         `json:"GvGMatch"`
	Participants         []KillPlayerResponse `json:"Participants"`
	GroupMembers         []KillPlayerResponse `json:"GroupMembers"`
}

type AlbionAPI struct {
	baseUrl    string
	client     *http.Client
	timeout    time.Duration
	maxRetries int
}

func NewAlbionAPI() *AlbionAPI {
	return &AlbionAPI{
		baseUrl:    "https://gameinfo.albiononline.com/api/gameinfo",
		client:     &http.Client{},
		timeout:    30 * time.Second,
		maxRetries: 3,
	}
}

func (a *AlbionAPI) FetchRecentBattles(offset, limit int) ([]BattleResponse, error) {
	// Use a random UUID to prevent caching
	url := fmt.Sprintf("%s/battles?offset=%d&limit=%d&sort=recent&guid=%s", a.baseUrl, offset, limit, uuid.New().String())
	var resp []BattleResponse
	if err := a.makeHttpGETCall(url, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (a *AlbionAPI) FetchBattle(battleId string) (*BattleResponse, error) {
	url := fmt.Sprintf("%s/battles/%s", a.baseUrl, battleId)
	var resp BattleResponse
	if err := a.makeHttpGETCall(url, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (a *AlbionAPI) FetchBattleKills(battleId, offset, limit int) ([]BattleKillResponse, error) {
	url := fmt.Sprintf("%s/events/battle/%d?offset=%d&limit=%d", a.baseUrl, battleId, offset, limit)
	var resp []BattleKillResponse
	if err := a.makeHttpGETCall(url, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// maxPagesToFetch limits pagination to prevent infinite fetching on empty database
const maxPagesToFetch = 5

// FetchRecentKillsUntilOverlap fetches kills, paginating only when ALL results are new.
// existingIds is a set of event IDs that already exist in the database (for fast lookup).
// This ensures we catch up if we've fallen behind, but don't unnecessarily paginate.
// Uses a consistent GUID across all pages and retries.
// Limited to maxPagesToFetch pages to prevent infinite pagination on empty DB.
func (a *AlbionAPI) FetchRecentKillsUntilOverlap(pageSize int, existingIds map[int]bool) ([]KillResponse, error) {
	guid := uuid.New().String()
	allKills := make([]KillResponse, 0)
	offset := 0
	page := 0

	for {
		// Safety limit to prevent infinite pagination (e.g., on empty database)
		if page >= maxPagesToFetch {
			fmt.Printf("Reached max pages limit (%d) - stopping pagination\n", maxPagesToFetch)
			break
		}

		kills, err := a.fetchKillsPage(offset, pageSize, guid)
		if err != nil {
			// Return what we have so far - partial results are better than none
			return allKills, fmt.Errorf("failed at offset %d: %w", offset, err)
		}

		if len(kills) == 0 {
			break
		}

		allKills = append(allKills, kills...)

		// Check how many of these kills already exist (in-memory lookup)
		newCount := 0
		for _, kill := range kills {
			if !existingIds[kill.EventId] {
				newCount++
			}
		}

		fmt.Printf("Page %d (offset %d): %d kills fetched, %d new, %d existing\n", page+1, offset, len(kills), newCount, len(kills)-newCount)

		// If we found ANY existing kills, we've caught up - stop paginating
		if newCount < len(kills) {
			break
		}

		// All kills are new - we might have fallen behind, fetch next page
		// But also stop if we got fewer results than requested (end of data)
		if len(kills) < pageSize {
			break
		}

		offset += pageSize
		page++
		fmt.Printf("All %d kills were new - fetching next page to catch up...\n", len(kills))
	}

	return allKills, nil
}

// FetchRecentKills fetches a single page of recent kills
func (a *AlbionAPI) FetchRecentKills(offset, limit int) ([]KillResponse, error) {
	guid := uuid.New().String()
	return a.fetchKillsPage(offset, limit, guid)
}

// fetchKillsPage fetches a single page of kills with retry logic
func (a *AlbionAPI) fetchKillsPage(offset, limit int, guid string) ([]KillResponse, error) {
	url := fmt.Sprintf("%s/events?offset=%d&limit=%d&guid=%s", a.baseUrl, offset, limit, guid)
	var resp []KillResponse
	if err := a.makeHttpGETCallWithRetry(url, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (a *AlbionAPI) makeHttpGETCall(url string, v interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	fmt.Printf("GET %s\n", url)
	resp, err := a.client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("request timed out after %s: %w", a.timeout, err)
		}
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %s, %d: %s", url, resp.StatusCode, string(body))
	}

	fmt.Println("GET response OK", url)
	return json.NewDecoder(resp.Body).Decode(v)
}

// makeHttpGETCallWithRetry performs a GET request with exponential backoff retry
func (a *AlbionAPI) makeHttpGETCallWithRetry(url string, v interface{}) error {
	var lastErr error

	for attempt := 1; attempt <= a.maxRetries; attempt++ {
		err := a.makeHttpGETCall(url, v)
		if err == nil {
			return nil
		}

		lastErr = err

		if attempt < a.maxRetries {
			// Exponential backoff: 2s, 4s, 8s...
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			fmt.Printf("Attempt %d failed: %v. Retrying in %v...\n", attempt, err, backoff)
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", a.maxRetries, lastErr)
}
