package main

import (
	"fmt"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"sort"
	"strconv"
)

type Battleboards struct {
	albionAPI *AlbionAPI
	app       *pocketbase.PocketBase
}

func NewBattleboards(app *pocketbase.PocketBase) *Battleboards {
	return &Battleboards{
		app:       app,
		albionAPI: NewAlbionAPI(),
	}
}

//func (b *Battleboards) processBattles(battles []BattleResponse) error {
//	// TODO: A queue would be ideal here
//	for _, battle := range battles {
//		if err := b.processBattle(battle); err != nil {
//			return err
//		}
//	}
//}

func (b *Battleboards) TempTest() error {
	lastBattleId, err := b.getLastBattleEnqueued()
	fmt.Println("Last enqueued battle ID:", lastBattleId)
	if err != nil {
		return err
	}

	battles, err := b.albionAPI.FetchRecentBattles(0, 51)
	if err != nil {
		return err
	}

	collection, err := b.app.FindCollectionByNameOrId("battle_queue")
	if err != nil {
		return err
	}

	// TODO: Pagination
	// at the moment, this will run on a CRON every minute
	// and we're assuming that the number of battles per minute is low enough to not require pagination
	records := make([]*core.Record, 0, len(battles))
	for _, battle := range battles {
		if strconv.Itoa(battle.Id) == lastBattleId {
			fmt.Println("Reached last enqueued battle:", lastBattleId)
			break // We've already processed up to this battle
		}
		record := mapBattleQueue(collection, battle)
		records = append(records, record)
	}

	if len(records) == 0 {
		fmt.Println("No new battles to enqueue.")
		return nil
	}

	err = b.app.RunInTransaction(func(txApp core.App) error {
		for _, record := range records {
			if err = txApp.Save(record); err != nil {
				return err
			}
		}
		return nil
	})

	fmt.Println("Enqueued", len(records), "new battles for processing.")
	return err
}

func (b *Battleboards) getLastBattleEnqueued() (string, error) {
	lastBattleInQueue, err := b.app.FindRecordsByFilter(
		"battle_queue",
		"",
		"-startTime",
		1,
		0)

	if err != nil {
		return "", err
	}

	lastBattleId := ""
	if len(lastBattleInQueue) > 0 {
		lastBattleId = lastBattleInQueue[0].Get("battleId").(string)
	}
	return lastBattleId, nil
}

func mapBattleQueue(collection *core.Collection, battle BattleResponse) *core.Record {
	record := core.NewRecord(collection)
	record.Set("battleId", strconv.Itoa(battle.Id))
	record.Set("region", "americas")
	record.Set("status", "queued")
	record.Set("startTime", battle.StartTime)
	return record
}

func (b *Battleboards) processBattle(battle BattleResponse) error {
	limit := 50
	offset := 0

	allKills := make([]KillsResponse, 0, battle.TotalKills)
	for offset < battle.TotalKills {
		kills, err := b.albionAPI.FetchRecentKills(battle.Id, offset, limit)
		if err != nil {
			return err
		}

		fmt.Println("Kills length", len(kills))

		allKills = append(allKills, kills...)
		offset += limit
	}

	fmt.Println("battle kills: ", battle.TotalKills)
	fmt.Println("Allkills length: ", len(allKills))

	//var battleParticipantsAlliances = make([]core.Record, 0, len(battle.Alliances))
	//var battleParticipantsGuilds = make([]core.Record, 0, len(battle.Guilds))
	//var battleParticipantsPlayers = make([]core.Record, 0, len(battle.Players))
	//
	//err := b.app.RunInTransaction(func(txApp core.App) error {
	//	return nil
	//})

	battleRecord, err := b.mapBattle(battle, allKills)
	if err != nil {
		return err
	}

	err = b.app.Save(battleRecord)
	if err != nil {
		if err.Error() != "id: Value must be unique." {
			return err
		}
	}

	return nil
}

type count struct {
	Name  string
	Count int
}

func (b *Battleboards) mapBattle(battle BattleResponse, allKills []KillsResponse) (*core.Record, error) {
	collection, err := b.app.FindCollectionByNameOrId("battles")
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(collection)
	record.Set("id", battle.Id)
	record.Set("region", "Americas")
	record.Set("startTime", battle.StartTime)
	record.Set("endTime", battle.EndTime)
	record.Set("totalFame", battle.TotalFame)
	record.Set("totalKills", battle.TotalKills)
	record.Set("numPlayers", len(battle.Players))

	guilds := getTopGuildsByParticipation(allKills)
	record.Set("guilds", guilds)

	alliances := getTopAlliancesByParticipation(allKills)
	record.Set("alliances", alliances)

	return record, nil
}

func getTopGuildsByParticipation(allKills []KillsResponse) string {
	guilds := mapGuildMembers(allKills)
	guildCounts := make([]count, 0, len(guilds))
	for name, members := range guilds {
		guildCounts = append(guildCounts, count{
			Name:  name,
			Count: len(members),
		})
	}
	topGuilds := getTopNCounts(guildCounts, 10)
	topGuildsStr := ""
	for _, guild := range topGuilds {
		if guild.Name == "" {
			continue
		}
		if topGuildsStr != "" {
			topGuildsStr += ", "
		}
		topGuildsStr += guild.Name
	}

	return topGuildsStr
}

func getTopAlliancesByParticipation(allKills []KillsResponse) string {
	alliances := mapAllianceMembers(allKills)
	allianceCounts := make([]count, 0, len(alliances))
	for name, members := range alliances {
		allianceCounts = append(allianceCounts, count{
			Name:  name,
			Count: len(members),
		})
	}
	topAlliances := getTopNCounts(allianceCounts, 10)
	topAlliancesStr := ""
	for _, alliance := range topAlliances {
		if alliance.Name == "" {
			continue
		}

		if topAlliancesStr != "" {
			topAlliancesStr += ", "
		}
		topAlliancesStr += alliance.Name
	}

	return topAlliancesStr
}

func getTopNCounts(counts []count, n int) []count {
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].Count > counts[j].Count
	})

	fmt.Println(counts)

	if len(counts) <= n {
		n = len(counts) - 1
	}

	return counts[:n+1]
}

func mapGuildMembers(allKills []KillsResponse) map[string]map[string]KillsPlayerResponse {
	guilds := make(map[string]map[string]KillsPlayerResponse)
	for _, kills := range allKills {
		for _, player := range append(kills.GroupMembers, kills.Participants...) {
			if player.GuildName == "" {
				continue
			}
			if _, exists := guilds[player.GuildName]; !exists {
				guilds[player.GuildName] = make(map[string]KillsPlayerResponse)
			}
			guilds[player.GuildName][player.Name] = player
		}
	}
	return guilds
}

func mapAllianceMembers(allKills []KillsResponse) map[string]map[string]KillsPlayerResponse {
	alliances := make(map[string]map[string]KillsPlayerResponse)
	for _, kills := range allKills {
		for _, player := range append(kills.GroupMembers, kills.Participants...) {
			if player.AllianceName == "" {
				continue
			}
			if _, exists := alliances[player.AllianceName]; !exists {
				alliances[player.AllianceName] = make(map[string]KillsPlayerResponse)
			}
			alliances[player.AllianceName][player.Name] = player
		}
	}
	return alliances
}
