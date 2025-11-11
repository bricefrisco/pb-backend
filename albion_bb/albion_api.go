package albion_bb

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"time"
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

type KillsItemResponse struct {
	Type    string `json:"Type"`
	Quality int    `json:"Quality"`
}

type KillsEquipmentResponse struct {
	MainHand KillsItemResponse `json:"MainHand"`
}

type KillsPlayerResponse struct {
	Id                 string                 `json:"Id"`
	Name               string                 `json:"Name"`
	AllianceId         string                 `json:"AllianceId"`
	AllianceName       string                 `json:"AllianceName"`
	GuildId            string                 `json:"GuildId"`
	GuildName          string                 `json:"GuildName"`
	KillFame           int                    `json:"KillFame"`
	DeathFame          int                    `json:"DeathFame"`
	AverageItemPower   float64                `json:"AverageItemPower"`
	DamageDone         float64                `json:"DamageDone"`
	SupportHealingDone float64                `json:"SupportHealingDone"`
	Equipment          KillsEquipmentResponse `json:"Equipment"`
}

type KillsResponse struct {
	BattleId            int                   `json:"BattleId"`
	Timestamp           time.Time             `json:"Timestamp"`
	Killer              KillsPlayerResponse   `json:"Killer"`
	Victim              KillsPlayerResponse   `json:"Victim"`
	TotalVictimKillFame int                   `json:"TotalVictimKillFame"`
	GroupMembers        []KillsPlayerResponse `json:"GroupMembers"`
	Participants        []KillsPlayerResponse `json:"Participants"`
}

type AlbionAPI struct {
	baseUrl string
	client  *http.Client
	retries int
}

func NewAlbionAPI() *AlbionAPI {
	return &AlbionAPI{
		baseUrl: "https://gameinfo.albiononline.com/api/gameinfo",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		retries: 3,
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

func (a *AlbionAPI) FetchRecentKills(battleId, offset, limit int) ([]KillsResponse, error) {
	url := fmt.Sprintf("%s/events/battle/%d?offset=%d&limit=%d", a.baseUrl, battleId, offset, limit)
	var resp []KillsResponse
	if err := a.makeHttpGETCall(url, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (a *AlbionAPI) makeHttpGETCall(url string, v interface{}) error {
	var lastErr error
	for i := 1; i <= a.retries; i++ {
		fmt.Printf("GET %s (attempt %d)\n", url, i)
		resp, err := a.client.Get(url)
		if err != nil {
			lastErr = err
		} else {
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				fmt.Println("GET response OK", url)
				return json.NewDecoder(resp.Body).Decode(v)
			}

			body, _ := io.ReadAll(resp.Body)
			errorMessage := fmt.Sprintf("API error %s, %d: %s", url, resp.StatusCode, string(body))
			fmt.Println(errorMessage)
			lastErr = fmt.Errorf(errorMessage)
		}

		if i < a.retries {
			sleep := time.Duration(i) * 15 * time.Second // simple linear backoff
			fmt.Println("Retrying in", sleep, "...")
			time.Sleep(sleep)
		}
	}

	return fmt.Errorf("failed after %d retries: %w", a.retries, lastErr)
}
