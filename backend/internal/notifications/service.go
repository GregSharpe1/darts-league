package notifications

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/slack"
)

type MessagePoster interface {
	PostMessage(ctx context.Context, channelID, text string) error
}

type RegistrationNotifier struct {
	poster    MessagePoster
	channelID string
	location  *time.Location
	logger    *log.Logger
}

func NewRegistrationNotifier(poster MessagePoster, channelID string, location *time.Location, logger *log.Logger) league.RegistrationNotifier {
	if location == nil {
		location = time.UTC
	}
	if logger == nil {
		logger = log.Default()
	}

	return RegistrationNotifier{
		poster:    poster,
		channelID: strings.TrimSpace(channelID),
		location:  location,
		logger:    logger,
	}
}

func (n RegistrationNotifier) NotifyPlayerRegistered(ctx context.Context, player league.Player, totalRegistered int) {
	if n.poster == nil || n.channelID == "" {
		return
	}

	text := fmt.Sprintf("New player signup\n- Player: %s\n- Signed up: %s\n- Total registered: %d", player.AdminLabel(), player.RegisteredAt.In(n.location).Format("Mon 02 Jan 2006 15:04 MST"), totalRegistered)
	if err := n.poster.PostMessage(ctx, n.channelID, text); err != nil && !errorsIsDisabled(err) {
		n.logger.Printf("slack signup notification failed: %v", err)
	}
}

type WeeklyService struct {
	store           league.Store
	now             func() time.Time
	poster          MessagePoster
	publicChannelID string
}

func NewWeeklyService(store league.Store, now func() time.Time, poster MessagePoster, publicChannelID string) WeeklyService {
	return WeeklyService{
		store:           store,
		now:             now,
		poster:          poster,
		publicChannelID: strings.TrimSpace(publicChannelID),
	}
}

func (s WeeklyService) PostWeeklyFixtures(ctx context.Context) (bool, error) {
	text, ok, err := s.ComposeWeeklyFixturesMessage(ctx)
	if err != nil || !ok {
		return false, err
	}
	if s.poster == nil || s.publicChannelID == "" {
		return false, nil
	}
	if err := s.poster.PostMessage(ctx, s.publicChannelID, text); err != nil {
		if errorsIsDisabled(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s WeeklyService) PostWeeklySummary(ctx context.Context) (bool, error) {
	text, ok, err := s.ComposeWeeklySummaryMessage(ctx)
	if err != nil || !ok {
		return false, err
	}
	if s.poster == nil || s.publicChannelID == "" {
		return false, nil
	}
	if err := s.poster.PostMessage(ctx, s.publicChannelID, text); err != nil {
		if errorsIsDisabled(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s WeeklyService) ComposeWeeklyFixturesMessage(ctx context.Context) (string, bool, error) {
	data, ok, err := s.weeklyData(ctx)
	if err != nil || !ok {
		return "", false, err
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("🎯 Week %d Fixtures\n", data.currentWeek))
	builder.WriteString(fmt.Sprintf("📅 %s • %s\n", data.week.Fixtures[0].ScheduledAt.In(data.location).Format("Mon 02 Jan 2006 15:04 MST"), data.season.Name))
	builder.WriteString("\n")
	for _, fixture := range data.week.Fixtures {
		builder.WriteString(fmt.Sprintf("🏆 %s vs %s\n",
			data.playersByID[fixture.PlayerOneID].PreferredName(),
			data.playersByID[fixture.PlayerTwoID].PreferredName(),
		))
	}

	return strings.TrimSpace(builder.String()), true, nil
}

func (s WeeklyService) ComposeWeeklySummaryMessage(ctx context.Context) (string, bool, error) {
	data, ok, err := s.weeklyData(ctx)
	if err != nil || !ok {
		return "", false, err
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("📣 Week %d Results + Standings\n", data.currentWeek))
	builder.WriteString(fmt.Sprintf("📅 %s • %s\n\n", data.now.In(data.location).Format("Fri 02 Jan 2006"), data.season.Name))
	builder.WriteString("✅ Results\n")

	resultsPosted := 0
	pendingResults := make([]string, 0)
	for _, fixture := range data.week.Fixtures {
		playerOne := data.playersByID[fixture.PlayerOneID].PreferredName()
		playerTwo := data.playersByID[fixture.PlayerTwoID].PreferredName()
		result, ok := data.resultsByFixtureID[fixture.ID]
		if !ok {
			pendingResults = append(pendingResults, fmt.Sprintf("- %s vs %s", playerOne, playerTwo))
			continue
		}
		resultsPosted++
		builder.WriteString(fmt.Sprintf("- %s %d-%d %s\n", playerOne, result.PlayerOneLegs, result.PlayerTwoLegs, playerTwo))
	}

	if resultsPosted == 0 {
		builder.WriteString("- No results recorded yet\n")
	}

	standings := league.BuildStandings(data.players, data.fixtures, data.results)
	if len(standings) > 0 {
		builder.WriteString(fmt.Sprintf("\n👑 Leader: %s on %d pts\n", standings[0].PreferredName, standings[0].Points))
	}
	if len(pendingResults) > 0 {
		builder.WriteString("\n⏳ Awaiting result\n")
		for _, line := range pendingResults {
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}

	builder.WriteString("\n📊 Standings\n```")
	builder.WriteString(formatStandingsTable(standings))
	builder.WriteString("```")

	return strings.TrimSpace(builder.String()), true, nil
}

type weeklyMessageData struct {
	season             league.Season
	players            []league.Player
	playersByID        map[int64]league.Player
	fixtures           []league.Fixture
	results            []league.Result
	resultsByFixtureID map[int64]league.Result
	week               league.WeeklyFixtures
	currentWeek        int
	location           *time.Location
	now                time.Time
}

func (s WeeklyService) weeklyData(ctx context.Context) (weeklyMessageData, bool, error) {
	if s.store == nil {
		return weeklyMessageData{}, false, nil
	}

	season, err := s.store.GetActiveSeason(ctx)
	if err != nil {
		return weeklyMessageData{}, false, err
	}
	fixtures, err := s.store.ListFixturesBySeason(ctx, season.ID)
	if err != nil {
		return weeklyMessageData{}, false, err
	}
	if len(fixtures) == 0 {
		return weeklyMessageData{}, false, nil
	}

	players, err := s.store.ListPlayersBySeason(ctx, season.ID)
	if err != nil {
		return weeklyMessageData{}, false, err
	}
	results, err := s.store.ListResultsBySeason(ctx, season.ID)
	if err != nil {
		return weeklyMessageData{}, false, err
	}

	location, err := time.LoadLocation(season.Timezone)
	if err != nil {
		return weeklyMessageData{}, false, err
	}
	now := s.now()
	currentWeek := league.CurrentPublicWeek(fixtures, now, location)
	if currentWeek == 0 {
		return weeklyMessageData{}, false, nil
	}

	grouped := league.GroupFixturesByWeek(fixtures, location)
	var current league.WeeklyFixtures
	for _, week := range grouped {
		if week.WeekNumber == currentWeek {
			current = week
			break
		}
	}
	if current.WeekNumber == 0 {
		return weeklyMessageData{}, false, nil
	}

	playersByID := make(map[int64]league.Player, len(players))
	for _, player := range players {
		playersByID[player.ID] = player
	}
	resultsByFixtureID := make(map[int64]league.Result, len(results))
	for _, result := range results {
		resultsByFixtureID[result.FixtureID] = result
	}

	return weeklyMessageData{
		season:             season,
		players:            players,
		playersByID:        playersByID,
		fixtures:           fixtures,
		results:            results,
		resultsByFixtureID: resultsByFixtureID,
		week:               current,
		currentWeek:        currentWeek,
		location:           location,
		now:                now,
	}, true, nil
}

func formatStandingsTable(rows []league.StandingRow) string {
	if len(rows) == 0 {
		return "No standings available\n"
	}

	labelWidth := len("Player")
	for _, row := range rows {
		if len(row.PreferredName) > labelWidth {
			labelWidth = len(row.PreferredName)
		}
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%-3s %-*s %2s %2s %2s %3s %3s %3s %3s\n", "#", labelWidth, "Player", "P", "W", "L", "LF", "LA", "LD", "Pts"))
	for index, row := range rows {
		builder.WriteString(fmt.Sprintf("%-3d %-*s %2d %2d %2d %3d %3d %3s %3d\n",
			index+1,
			labelWidth,
			row.PreferredName,
			row.Played,
			row.Won,
			row.Lost,
			row.LegsFor,
			row.LegsAgainst,
			formatLegDifference(row.LegDifference),
			row.Points,
		))
	}

	return builder.String()
}

func formatLegDifference(value int) string {
	if value > 0 {
		return fmt.Sprintf("+%d", value)
	}
	return fmt.Sprintf("%d", value)
}

func errorsIsDisabled(err error) bool {
	return errors.Is(err, slack.ErrDisabled)
}

func WeeklyCommandName(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return strings.TrimSpace(args[0])
}
