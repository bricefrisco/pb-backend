package main

import (
	"fmt"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
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

func (b *Battleboards) EnqueueNewBattles() error {
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

func (b *Battleboards) ProcessBattlesInQueue() error {
	enqueuedBattlesToProcess, err := b.app.FindRecordsByFilter(
		"battle_queue",
		"status = 'queued' || status = 'failed'",
		"-startTime",
		1,
		0)
	if err != nil {
		return err
	}

	for _, record := range enqueuedBattlesToProcess {
		// TODO: Multithreading
		queueId := record.Get("id").(string)
		battleId := record.Get("battleId").(string)
		err = b.processBattle(queueId, battleId)
		if err != nil {
			fmt.Println("Error processing battle", battleId, ":", err)
			continue
		}
	}

	return nil
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

func (b *Battleboards) processBattle(queueId string, battleId string) error {
	fmt.Println("Processing battle:", battleId)

	battle, err := b.albionAPI.FetchBattle(battleId)
	if err != nil {
		return err
	}

	limit := 50
	offset := 0

	allKills := make([]KillsResponse, 0)
	for offset < battle.TotalKills {
		kills, err := b.albionAPI.FetchRecentKills(battle.Id, offset, limit)
		if err != nil {
			return err
		}

		allKills = append(allKills, kills...)
		offset += limit
	}

	allianceInputData := make([]*AllianceInputData, 0)
	for _, alliance := range battle.Alliances {
		allianceInputData = append(allianceInputData, &AllianceInputData{
			Id:   alliance.Id,
			Name: alliance.Name,
		})
	}

	allianceData := mapAllianceData(allianceInputData, allKills)
	for _, data := range allianceData {
		fmt.Printf("Alliance: %+v\n", data)
	}

	guildInputData := make([]*GuildInputData, 0)
	for _, guild := range battle.Guilds {
		guildInputData = append(guildInputData, &GuildInputData{
			Id:           guild.Id,
			Name:         guild.Name,
			AllianceId:   guild.AllianceId,
			AllianceName: guild.AllianceName,
		})
	}

	guildData := mapGuildData(guildInputData, allKills)
	for _, data := range guildData {
		fmt.Printf("Guild: %+v\n", data)
	}

	numPlayers := getTotalPlayers(allKills)

	battleRecord, err := b.mapBattle(battle, allianceData, guildData, numPlayers)
	if err != nil {
		return err
	}

	allianceRecords, err := b.mapAlliances(battle.Id, allianceData)
	if err != nil {
		return err
	}

	guildRecords, err := b.mapGuilds(battle.Id, guildData)
	if err != nil {
		return err
	}

	queue, err := b.app.FindRecordById("battle_queue", queueId)
	if err != nil {
		return err
	}

	err = b.app.RunInTransaction(func(txApp core.App) error {
		err = txApp.Save(battleRecord)
		if err != nil {
			return err
		}

		for _, record := range allianceRecords {
			err = txApp.Save(record)
			if err != nil {
				return err
			}
		}

		for _, record := range guildRecords {
			err = txApp.Save(record)
			if err != nil {
				return err
			}
		}

		queue.Set("status", "processed")
		err = txApp.Save(queue)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error processing battle %s: %v\n", battleId, err)
		queue.Set("status", "failed")
		err = b.app.Save(queue)
		if err != nil {
			// Unlikely, but the status would be left in the 'processing' state
			fmt.Println("Also failed to update battle queue status to 'failed':", err)
			return err
		}
	} else {
		fmt.Println("Successfully processed battle:", battleId)
	}

	return nil
}

func (b *Battleboards) mapBattle(battle *BattleResponse, allianceData []*AllianceData, guildData []*GuildData, numPlayers int) (*core.Record, error) {
	collection, err := b.app.FindCollectionByNameOrId("battles")
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(collection)
	record.Set("id", battle.Id)
	record.Set("region", "americas")
	record.Set("startTime", battle.StartTime)
	record.Set("endTime", battle.EndTime)
	record.Set("totalFame", battle.TotalFame)
	record.Set("totalKills", battle.TotalKills)
	record.Set("numPlayers", numPlayers)

	alliances := getTopAlliancesByParticipation(allianceData)
	record.Set("alliances", alliances)

	guilds := getTopGuildsByParticipation(guildData)
	record.Set("guilds", guilds)

	return record, nil
}

func (b *Battleboards) mapAlliances(battleId int, allianceData []*AllianceData) ([]*core.Record, error) {
	collection, err := b.app.FindCollectionByNameOrId("battle_participants_alliances")
	if err != nil {
		return nil, err
	}

	records := make([]*core.Record, 0, len(allianceData))
	for _, alliance := range allianceData {
		record := core.NewRecord(collection)
		record.Set("battle", battleId)
		record.Set("region", "americas")
		record.Set("allianceId", alliance.Id)
		record.Set("allianceName", alliance.Name)
		record.Set("kills", alliance.Kills)
		record.Set("killFame", alliance.KillFame)
		record.Set("deaths", alliance.Deaths)
		record.Set("deathFame", alliance.DeathFame)
		record.Set("players", alliance.Players)
		record.Set("averageIp", alliance.AverageIp)
		records = append(records, record)
	}

	return records, nil
}

func (b *Battleboards) mapGuilds(battleId int, guildData []*GuildData) ([]*core.Record, error) {
	collection, err := b.app.FindCollectionByNameOrId("battle_participants_guilds")
	if err != nil {
		return nil, err
	}

	records := make([]*core.Record, 0, len(guildData))
	for _, guild := range guildData {
		record := core.NewRecord(collection)
		record.Set("battle", battleId)
		record.Set("region", "americas")
		record.Set("guildId", guild.Id)
		record.Set("guildName", guild.Name)
		record.Set("allianceId", guild.AllianceId)
		record.Set("allianceName", guild.AllianceName)
		record.Set("kills", guild.Kills)
		record.Set("killFame", guild.KillFame)
		record.Set("deaths", guild.Deaths)
		record.Set("deathFame", guild.DeathFame)
		record.Set("players", guild.Players)
		record.Set("averageIp", guild.AverageIp)
		records = append(records, record)
	}

	return records, nil
}
