package main

import (
	"fmt"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"strconv"
	"strings"
)

type queueItem struct {
	queueId  string
	battleId string
}

type Battleboards struct {
	albionAPI     *AlbionAPI
	app           *pocketbase.PocketBase
	queue         chan queueItem
	maxIterations int
}

func NewBattleboards(app *pocketbase.PocketBase) *Battleboards {
	return &Battleboards{
		app:           app,
		albionAPI:     NewAlbionAPI(),
		queue:         make(chan queueItem, 100),
		maxIterations: 100,
	}
}

func (b *Battleboards) FetchNewBattles() error {
	lastBattleId, err := b.getLastBattleFetched()
	fmt.Println("Last fetched battle ID:", lastBattleId)
	if err != nil {
		return err
	}

	collection, err := b.app.FindCollectionByNameOrId("battle_queue")
	if err != nil {
		return err
	}

	reachedLastBattle := false
	iteration := 0
	records := make([]*core.Record, 0)

	for !reachedLastBattle && iteration < b.maxIterations {
		battles, err := b.albionAPI.FetchRecentBattles(iteration*50, 50)
		if err != nil {
			return err
		}

		for _, battle := range battles {
			if strconv.Itoa(battle.Id) == lastBattleId {
				fmt.Println("Reached last fetched battle:", lastBattleId)
				reachedLastBattle = true
				break // We've already processed up to this battle
			}
			record := mapBattleQueue(collection, battle)
			records = append(records, record)
		}

		iteration += 1
	}

	if len(records) == 0 {
		fmt.Println("No new battles fetched.")
		return nil
	}

	err = b.app.RunInTransaction(func(txApp core.App) error {
		ids := make([]string, 0, len(records))
		for _, record := range records {
			ids = append(ids, record.Get("battleId").(string))
		}

		recs, err := txApp.FindRecordsByIds("battle_queue", ids)
		if err != nil {
			return err
		}

		existingIds := make(map[string]bool)
		for _, rec := range recs {
			existingIds[rec.Get("battleId").(string)] = true
		}

		for _, record := range records {
			if _, exists := existingIds[record.Get("battleId").(string)]; exists {
				fmt.Println("Battle already in queue, skipping:", record.Get("battleId").(string))
				continue
			}

			if err = txApp.Save(record); err != nil {
				return err
			}
		}
		return nil
	})

	fmt.Println("Fetched", len(records), "new battles for processing.")
	return err
}

func (b *Battleboards) EnqueueNewBattles() error {
	enqueuedBattlesToProcess, err := b.app.FindRecordsByFilter(
		"battle_queue",
		"status = 'queued' || status = 'failed'",
		"-startTime",
		100,
		0)

	if err != nil {
		return err
	}

	if len(enqueuedBattlesToProcess) == 0 {
		fmt.Println("No battles to enqueue.")
		return nil
	}

	fmt.Println("Enqueuing", len(enqueuedBattlesToProcess), "battles for processing...")
	go func() {
		for _, record := range enqueuedBattlesToProcess {
			queueId := record.Get("id").(string)
			battleId := record.Get("battleId").(string)

			b.queue <- queueItem{
				queueId:  queueId,
				battleId: battleId,
			}
		}
	}()

	return nil
}

func (b *Battleboards) ProcessQueue() {
	fmt.Println("Starting battle processing queue...")
	for job := range b.queue {
		err := b.processBattle(job.queueId, job.battleId)
		if err != nil {
			fmt.Println("Error processing battle", job.battleId, ":", err)
			continue
		}
	}
}

func (b *Battleboards) getLastBattleFetched() (string, error) {
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

	queue, err := b.app.FindRecordById("battle_queue", queueId)
	if err != nil {
		return err
	}

	queue.Set("status", "processing")
	err = b.app.Save(queue)

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

	guildInputData := make([]*GuildInputData, 0)
	for _, guild := range battle.Guilds {
		guildInputData = append(guildInputData, &GuildInputData{
			Id:           guild.Id,
			Name:         guild.Name,
			AllianceId:   guild.AllianceId,
			AllianceName: guild.AllianceName,
		})
	}

	allianceData := mapAllianceData(allianceInputData, allKills)
	guildData := mapGuildData(guildInputData, allKills)
	playerData := mapPlayerData(allKills)
	numPlayers := len(playerData)

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

	playerRecords, err := b.mapPlayers(battle.Id, playerData)
	if err != nil {
		return err
	}

	kills, err := b.mapKills(battle.Id, allKills)
	if err != nil {
		return err
	}

	err = b.app.RunInTransaction(func(txApp core.App) error {
		fmt.Println("Saving battle record...")
		err = txApp.Save(battleRecord)
		if err != nil {
			return err
		}

		fmt.Println("Saving", len(allianceRecords), "alliance records...")
		for _, record := range allianceRecords {
			err = txApp.Save(record)
			if err != nil {
				return err
			}
		}

		fmt.Println("Saving", len(guildRecords), "guild records...")
		for _, record := range guildRecords {
			err = txApp.Save(record)
			if err != nil {
				return err
			}
		}

		fmt.Println("Saving", len(playerRecords), "player records...")
		for _, record := range playerRecords {
			err = txApp.Save(record)
			if err != nil {
				return err
			}
		}

		fmt.Println("Saving", len(kills), "kill records...")
		for _, record := range kills {
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
		if strings.Contains(err.Error(), "Value must be unique") {
			queue.Set("status", "processed")
		} else {
			queue.Set("status", "failed")
		}

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

func (b *Battleboards) mapPlayers(battleId int, playerData []*PlayerData) ([]*core.Record, error) {
	collection, err := b.app.FindCollectionByNameOrId("battle_participants_players")
	if err != nil {
		return nil, err
	}

	records := make([]*core.Record, 0, len(playerData))
	for _, player := range playerData {
		record := core.NewRecord(collection)
		record.Set("battle", battleId)
		record.Set("region", "americas")
		record.Set("playerId", player.Id)
		record.Set("playerName", player.Name)
		record.Set("allianceId", player.AllianceId)
		record.Set("allianceName", player.AllianceName)
		record.Set("guildId", player.GuildId)
		record.Set("guildName", player.GuildName)
		record.Set("kills", player.Kills)
		record.Set("killFame", player.KillFame)
		record.Set("deaths", player.Deaths)
		record.Set("deathFame", player.DeathFame)
		record.Set("weaponName", player.WeaponName)
		record.Set("averageIp", player.AverageIp)
		record.Set("damage", player.Damage)
		record.Set("healing", player.Healing)
		records = append(records, record)
	}
	return records, nil
}

func (b *Battleboards) mapKills(battleId int, kills []KillsResponse) ([]*core.Record, error) {
	collection, err := b.app.FindCollectionByNameOrId("battle_kills")
	if err != nil {
		return nil, err
	}

	records := make([]*core.Record, 0, len(kills))
	for _, kill := range kills {
		record := core.NewRecord(collection)
		record.Set("battle", battleId)
		record.Set("region", "americas")
		record.Set("timestamp", kill.Timestamp)

		record.Set("killerId", kill.Killer.Id)
		record.Set("killerName", kill.Killer.Name)
		record.Set("killerAllianceId", kill.Killer.AllianceId)
		record.Set("killerAllianceName", kill.Killer.AllianceName)
		record.Set("killerGuildId", kill.Killer.GuildId)
		record.Set("killerGuildName", kill.Killer.GuildName)
		record.Set("killerWeapon", kill.Killer.Equipment.MainHand.Type)
		record.Set("killerAverageIp", kill.Killer.AverageItemPower)

		record.Set("victimId", kill.Victim.Id)
		record.Set("victimName", kill.Victim.Name)
		record.Set("victimAllianceId", kill.Victim.AllianceId)
		record.Set("victimAllianceName", kill.Victim.AllianceName)
		record.Set("victimGuildId", kill.Victim.GuildId)
		record.Set("victimGuildName", kill.Victim.GuildName)
		record.Set("victimWeapon", kill.Victim.Equipment.MainHand.Type)
		record.Set("victimAverageIp", kill.Victim.AverageItemPower)
		records = append(records, record)
	}

	return records, nil
}
