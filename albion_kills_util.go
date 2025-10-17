package main

import (
	"sort"
)

type AllianceInputData struct {
	Id   string
	Name string
}

type AllianceData struct {
	Id        string
	Name      string
	Players   int
	Kills     int
	KillFame  int
	Deaths    int
	DeathFame int
	AverageIp float64
}

type GuildInputData struct {
	Id           string
	Name         string
	AllianceId   string
	AllianceName string
}

type GuildData struct {
	Id           string
	Name         string
	AllianceId   string
	AllianceName string
	Players      int
	Kills        int
	KillFame     int
	Deaths       int
	DeathFame    int
	AverageIp    float64
}

func mapAllianceData(alliances []*AllianceInputData, allKills []KillsResponse) []*AllianceData {
	result := make([]*AllianceData, 0)
	players := mapAlliancePlayers(allKills)
	killCounts := mapAllianceKillCounts(allKills)
	killFame := mapAllianceKillFame(allKills)
	deathCounts := mapAllianceDeathCounts(allKills)
	deathFame := mapAllianceDeathFame(allKills)

	for _, alliance := range alliances {
		result = append(result, &AllianceData{
			Id:        alliance.Id,
			Name:      alliance.Name,
			Players:   len(players[alliance.Name]),
			Kills:     killCounts[alliance.Name],
			KillFame:  killFame[alliance.Name],
			Deaths:    deathCounts[alliance.Name],
			DeathFame: deathFame[alliance.Name],
			AverageIp: mapAverageIp(players[alliance.Name]),
		})
	}

	return result
}

func mapGuildData(guilds []*GuildInputData, allKills []KillsResponse) []*GuildData {
	result := make([]*GuildData, 0)
	players := mapGuildPlayers(allKills)
	killCounts := mapGuildKillCounts(allKills)
	killFame := mapGuildKillFame(allKills)
	deathCounts := mapGuildDeathCounts(allKills)
	deathFame := mapGuildDeathFame(allKills)

	for _, guild := range guilds {
		result = append(result, &GuildData{
			Id:           guild.Id,
			Name:         guild.Name,
			AllianceId:   guild.AllianceId,
			AllianceName: guild.AllianceName,
			Players:      len(players[guild.Name]),
			Kills:        killCounts[guild.Name],
			KillFame:     killFame[guild.Name],
			Deaths:       deathCounts[guild.Name],
			DeathFame:    deathFame[guild.Name],
			AverageIp:    mapAverageIp(players[guild.Name]),
		})
	}

	return result
}

func mapAllianceKillCounts(allKills []KillsResponse) map[string]int {
	result := make(map[string]int)
	for _, kills := range allKills {
		if kills.Killer.AllianceName == "" {
			continue
		}

		_, exists := result[kills.Killer.AllianceName]
		if !exists {
			result[kills.Killer.AllianceName] = 0
		}
		result[kills.Killer.AllianceName]++

	}
	return result
}

func mapGuildKillCounts(allKills []KillsResponse) map[string]int {
	result := make(map[string]int)
	for _, kills := range allKills {
		if kills.Killer.GuildName == "" {
			continue
		}

		_, exists := result[kills.Killer.GuildName]
		if !exists {
			result[kills.Killer.GuildName] = 0
		}
		result[kills.Killer.GuildName]++

	}
	return result
}

func mapAllianceKillFame(allKills []KillsResponse) map[string]int {
	result := make(map[string]int)
	for _, kills := range allKills {
		if kills.Killer.AllianceName == "" {
			continue
		}

		_, exists := result[kills.Killer.AllianceName]
		if !exists {
			result[kills.Killer.AllianceName] = 0
		}
		result[kills.Killer.AllianceName] += kills.TotalVictimKillFame
	}
	return result
}

func mapGuildKillFame(allKills []KillsResponse) map[string]int {
	result := make(map[string]int)
	for _, kills := range allKills {
		if kills.Killer.GuildName == "" {
			continue
		}

		_, exists := result[kills.Killer.GuildName]
		if !exists {
			result[kills.Killer.GuildName] = 0
		}
		result[kills.Killer.GuildName] += kills.TotalVictimKillFame
	}
	return result
}

func mapAllianceDeathFame(allKills []KillsResponse) map[string]int {
	result := make(map[string]int)
	for _, kills := range allKills {
		if kills.Victim.AllianceName == "" {
			continue
		}

		_, exists := result[kills.Victim.AllianceName]
		if !exists {
			result[kills.Victim.AllianceName] = 0
		}
		result[kills.Victim.AllianceName] += kills.TotalVictimKillFame
	}
	return result
}

func mapGuildDeathFame(allKills []KillsResponse) map[string]int {
	result := make(map[string]int)
	for _, kills := range allKills {
		if kills.Victim.GuildName == "" {
			continue
		}

		_, exists := result[kills.Victim.GuildName]
		if !exists {
			result[kills.Victim.GuildName] = 0
		}
		result[kills.Victim.GuildName] += kills.TotalVictimKillFame
	}
	return result
}

func mapAllianceDeathCounts(allKills []KillsResponse) map[string]int {
	result := make(map[string]int)
	for _, kills := range allKills {
		if kills.Victim.AllianceName == "" {
			continue
		}

		_, exists := result[kills.Victim.AllianceName]
		if !exists {
			result[kills.Victim.AllianceName] = 0
		}
		result[kills.Victim.AllianceName]++

	}
	return result
}

func mapGuildDeathCounts(allKills []KillsResponse) map[string]int {
	result := make(map[string]int)
	for _, kills := range allKills {
		if kills.Victim.GuildName == "" {
			continue
		}

		_, exists := result[kills.Victim.GuildName]
		if !exists {
			result[kills.Victim.GuildName] = 0
		}
		result[kills.Victim.GuildName]++

	}
	return result
}

func mapAverageIp(players map[string]KillsPlayerResponse) float64 {
	count := 0
	sum := 0.0
	for _, player := range players {
		if player.AverageItemPower > 0 {
			count += 1
			sum += player.AverageItemPower
		}
	}

	if count == 0 {
		return 0.0
	}

	return sum / float64(count)
}

func mapAlliancePlayers(allKills []KillsResponse) map[string]map[string]KillsPlayerResponse {
	result := make(map[string]map[string]KillsPlayerResponse)
	for _, kills := range allKills {
		// Other battleboards (AlbionBB, official API) do not include participants in
		// total player count. This may cause a discrepancy, but it seems more accurate to include them.
		allPlayers := append(kills.GroupMembers, kills.Participants...)
		allPlayers = append(allPlayers, kills.Victim)
		allPlayers = append(allPlayers, kills.Killer)
		for _, player := range allPlayers {
			if player.AllianceName == "" {
				continue
			}
			if _, exists := result[player.AllianceName]; !exists {
				result[player.AllianceName] = make(map[string]KillsPlayerResponse)
			}

			// Participants stores item power while GroupMembers do not, so
			// we want to keep the entry that has the item power if available
			if existingPlayer, exists := result[player.AllianceName][player.Name]; exists {
				if existingPlayer.AverageItemPower < player.AverageItemPower {
					result[player.AllianceName][player.Name] = player
				}
			} else {
				result[player.AllianceName][player.Name] = player
			}
		}
	}
	return result
}

func mapGuildPlayers(allKills []KillsResponse) map[string]map[string]KillsPlayerResponse {
	result := make(map[string]map[string]KillsPlayerResponse)
	for _, kills := range allKills {
		// Other battleboards (AlbionBB, official API) do not include participants in
		// total player count. This may cause a discrepancy, but it seems more accurate to include them.
		allPlayers := append(kills.GroupMembers, kills.Participants...)
		allPlayers = append(allPlayers, kills.Victim)
		allPlayers = append(allPlayers, kills.Killer)
		for _, player := range allPlayers {
			if player.GuildName == "" {
				continue
			}
			if _, exists := result[player.GuildName]; !exists {
				result[player.GuildName] = make(map[string]KillsPlayerResponse)
			}

			// Participants stores item power while GroupMembers do not, so
			// we want to keep the entry that has the item power if available
			if existingPlayer, exists := result[player.GuildName][player.Name]; exists {
				if existingPlayer.AverageItemPower < player.AverageItemPower {
					result[player.GuildName][player.Name] = player
				}
			} else {
				result[player.GuildName][player.Name] = player
			}
		}
	}
	return result
}

func getTotalPlayers(allKills []KillsResponse) int {
	players := make(map[string]bool)
	for _, kills := range allKills {
		// Other battleboards (AlbionBB, official API) do not include participants in
		// total player count. This may cause a discrepancy, but it seems more accurate to include them.
		allPlayers := append(kills.GroupMembers, kills.Participants...)
		allPlayers = append(allPlayers, kills.Victim)
		allPlayers = append(allPlayers, kills.Killer)
		for _, player := range allPlayers {
			players[player.Name] = true
		}
	}
	return len(players)
}

func getTopAlliancesByParticipation(allianceData []*AllianceData) string {
	sort.Slice(allianceData, func(i, j int) bool {
		return allianceData[i].Players > allianceData[j].Players
	})

	result := ""
	for _, alliance := range allianceData {
		if result != "" {
			result += ", "
		}
		result += alliance.Name
	}

	return result
}

func getTopGuildsByParticipation(guildData []*GuildData) string {
	sort.Slice(guildData, func(i, j int) bool {
		return guildData[i].Players > guildData[j].Players
	})

	result := ""
	for _, guild := range guildData {
		if result != "" {
			result += ", "
		}
		result += guild.Name
	}

	return result
}
