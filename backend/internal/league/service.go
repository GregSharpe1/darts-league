package league

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"
)

var lockedFixturePlaceholders = [][2]string{
	{"I knew you'd look", "Nothing to see here"},
	{"Definitely not your match", "Move along, inspector"},
	{"Future you problem", "Try again Monday"},
	{"Nice try, dev", "These aren't the darts"},
}

var ErrSeasonNotFound = errors.New("active season not found")

type Store interface {
	EnsureActiveSeason(ctx context.Context, season Season) (Season, error)
	GetActiveSeason(ctx context.Context) (Season, error)
	ListPlayersBySeason(ctx context.Context, seasonID int64) ([]Player, error)
	AssignPlayer(ctx context.Context, seasonID, playerID int64, divisionID *int64, status PlayerStatus) (Player, error)
	ListDivisionsBySeason(ctx context.Context, seasonID int64) ([]Division, error)
	GetDivisionBySlug(ctx context.Context, seasonID int64, slug string) (Division, error)
	ReplaceDivisions(ctx context.Context, seasonID int64, divisions []Division) ([]Division, error)
	UpdateDivision(ctx context.Context, division Division) (Division, error)
	ListFixturesBySeason(ctx context.Context, seasonID int64) ([]Fixture, error)
	ListFixturesByDivision(ctx context.Context, divisionID int64) ([]Fixture, error)
	GetFixture(ctx context.Context, fixtureID int64) (Fixture, error)
	ListResultsBySeason(ctx context.Context, seasonID int64) ([]Result, error)
	ListResultsByDivision(ctx context.Context, divisionID int64) ([]Result, error)
	GetResultByFixture(ctx context.Context, fixtureID int64) (Result, error)
	ListAuditLogsBySeason(ctx context.Context, seasonID int64) ([]AuditLogEntry, error)
	ListAuditLogsByDivision(ctx context.Context, divisionID int64) ([]AuditLogEntry, error)
	CreatePlayer(ctx context.Context, player Player) (Player, error)
	CreateFixtures(ctx context.Context, fixtures []Fixture) ([]Fixture, error)
	ReplaceFixturesBySeason(ctx context.Context, seasonID int64, fixtures []Fixture) ([]Fixture, error)
	CreateResult(ctx context.Context, result Result) (Result, error)
	UpdateResult(ctx context.Context, result Result) (Result, error)
	DeleteResultByFixture(ctx context.Context, fixtureID int64) error
	CreateAuditLog(ctx context.Context, entry AuditLogEntry) (AuditLogEntry, error)
	DeletePlayer(ctx context.Context, seasonID, playerID int64) error
	UpsertSeason(ctx context.Context, season Season) (Season, error)
}

type RegistrationService struct {
	store    Store
	now      func() time.Time
	notifier RegistrationNotifier
}

type RegistrationNotifier interface {
	NotifyPlayerRegistered(ctx context.Context, player Player, totalRegistered int)
}

type noopRegistrationNotifier struct{}

func (noopRegistrationNotifier) NotifyPlayerRegistered(_ context.Context, _ Player, _ int) {}

func NewRegistrationService(store Store) RegistrationService {
	return NewRegistrationServiceWithNow(store, time.Now)
}

func NewRegistrationServiceWithNow(store Store, now func() time.Time) RegistrationService {
	return NewRegistrationServiceWithNowAndNotifier(store, now, noopRegistrationNotifier{})
}

func NewRegistrationServiceWithNowAndNotifier(store Store, now func() time.Time, notifier RegistrationNotifier) RegistrationService {
	if notifier == nil {
		notifier = noopRegistrationNotifier{}
	}

	return RegistrationService{
		store:    store,
		now:      now,
		notifier: notifier,
	}
}

func (s RegistrationService) RegisterPlayer(ctx context.Context, player Player) (Player, error) {
	season, existingPlayers, err := s.activeSeasonAndPlayers(ctx)
	if err != nil {
		return Player{}, err
	}

	book := NewRegistrationBook(existingPlayers)
	if err := book.ValidateNewPlayer(season, player); err != nil {
		return Player{}, err
	}

	player.SeasonID = season.ID
	player.DisplayName = normalizeSpacing(player.DisplayName)
	player.Nickname = normalizeSpacing(player.Nickname)
	player.Status = PlayerStatusWaitlist
	player.RegisteredAt = s.now().UTC()

	created, err := s.store.CreatePlayer(ctx, player)
	if err != nil {
		return Player{}, err
	}

	s.notifier.NotifyPlayerRegistered(ctx, created, len(existingPlayers)+1)

	return created, nil
}

func (s RegistrationService) ListPlayers(ctx context.Context) ([]Player, error) {
	season, players, err := s.activeSeasonAndPlayers(ctx)
	if err != nil {
		return nil, err
	}

	sorted := make([]Player, len(players))
	copy(sorted, players)
	sort.Slice(sorted, func(i, j int) bool {
		return NormalizeDisplayName(sorted[i].DisplayName) < NormalizeDisplayName(sorted[j].DisplayName)
	})

	for i := range sorted {
		sorted[i].SeasonID = season.ID
	}

	return sorted, nil
}

func (s RegistrationService) DeletePlayer(ctx context.Context, playerID int64) error {
	season, existingPlayers, err := s.activeSeasonAndPlayers(ctx)
	if err != nil {
		return err
	}

	book := NewRegistrationBook(existingPlayers)
	if err := book.ValidatePlayerDelete(season); err != nil {
		return err
	}

	return s.store.DeletePlayer(ctx, season.ID, playerID)
}

func (s RegistrationService) AssignPlayer(ctx context.Context, playerID int64, divisionID *int64) (Player, error) {
	season, players, err := s.activeSeasonAndPlayers(ctx)
	if err != nil {
		return Player{}, err
	}
	fixtures, err := s.store.ListFixturesBySeason(ctx, season.ID)
	if err != nil {
		return Player{}, err
	}
	adminLocked, err := NewSeasonServiceWithNow(s.store, s.now).adminLockedForSeason(season, fixtures)
	if err != nil {
		return Player{}, err
	}
	if adminLocked {
		return Player{}, ErrPlayerAssignLocked
	}
	if divisionID != nil {
		divisions, err := s.store.ListDivisionsBySeason(ctx, season.ID)
		if err != nil {
			return Player{}, err
		}
		found := false
		for _, division := range divisions {
			if division.ID == *divisionID {
				found = true
				break
			}
		}
		if !found {
			return Player{}, ErrDivisionNotFound
		}
	}
	status := PlayerStatusWaitlist
	if divisionID != nil {
		status = PlayerStatusAssigned
	}
	assignedPlayer, err := s.store.AssignPlayer(ctx, season.ID, playerID, divisionID, status)
	if err != nil {
		return Player{}, err
	}
	if season.RegistrationOpen() {
		return assignedPlayer, nil
	}
	updatedPlayers := make([]Player, len(players))
	copy(updatedPlayers, players)
	for index := range updatedPlayers {
		if updatedPlayers[index].ID == assignedPlayer.ID {
			updatedPlayers[index] = assignedPlayer
			break
		}
	}
	if err := rebuildSeasonFixtures(ctx, s.store, season, updatedPlayers); err != nil {
		return Player{}, err
	}
	return assignedPlayer, nil
}

func rebuildSeasonFixtures(ctx context.Context, store Store, season Season, players []Player) error {
	divisions, err := store.ListDivisionsBySeason(ctx, season.ID)
	if err != nil {
		return err
	}
	playersByDivision := make(map[int64][]Player, len(divisions))
	for _, player := range players {
		if player.DivisionID == nil || player.Status != PlayerStatusAssigned {
			continue
		}
		playersByDivision[*player.DivisionID] = append(playersByDivision[*player.DivisionID], player)
	}
	fixtures := make([]Fixture, 0)
	for _, division := range divisions {
		divisionPlayers := playersByDivision[division.ID]
		if len(divisionPlayers) < 2 {
			continue
		}
		divisionFixtures, err := GenerateRoundRobinFixtures(season, division, divisionPlayers)
		if err != nil {
			return err
		}
		fixtures = append(fixtures, divisionFixtures...)
	}
	_, err = store.ReplaceFixturesBySeason(ctx, season.ID, fixtures)
	return err
}

func (s RegistrationService) activeSeasonAndPlayers(ctx context.Context) (Season, []Player, error) {
	return activeSeasonAndPlayers(ctx, s.store)
}

func activeSeasonAndPlayers(ctx context.Context, store Store) (Season, []Player, error) {
	season, err := store.GetActiveSeason(ctx)
	if err != nil {
		return Season{}, nil, err
	}

	players, err := store.ListPlayersBySeason(ctx, season.ID)
	if err != nil {
		return Season{}, nil, err
	}

	return season, players, nil
}

func normalizeSpacing(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

type SeasonSummary struct {
	ID               int64
	Name             string
	Status           SeasonStatus
	Timezone         string
	StartedAt        *time.Time
	RegistrationOpen bool
	SeasonStarted    bool
	AdminLocked      bool
	CanStartSeason   bool
	CanEditSettings  bool
	CanEditDivisionChannel bool
	CanEditDivisions bool
	CanAssignPlayers bool
	PlayerCount      int
	WeekCount        int
	GameVariant      string
	LegsToWin        int
	GamesPerWeek     int
	TotalFixtures    int
	DivisionCount    int
	AssignedCount    int
	WaitlistCount    int
}

type SeasonService struct {
	store Store
	now   func() time.Time
}

func NewSeasonService(store Store) SeasonService {
	return NewSeasonServiceWithNow(store, time.Now)
}

func NewSeasonServiceWithNow(store Store, now func() time.Time) SeasonService {
	return SeasonService{store: store, now: now}
}

func (s SeasonService) Summary(ctx context.Context) (SeasonSummary, error) {
	season, err := s.store.GetActiveSeason(ctx)
	if err != nil {
		return SeasonSummary{}, err
	}

	players, err := s.store.ListPlayersBySeason(ctx, season.ID)
	if err != nil {
		return SeasonSummary{}, err
	}
	divisions, err := s.store.ListDivisionsBySeason(ctx, season.ID)
	if err != nil {
		return SeasonSummary{}, err
	}

	fixtures, err := s.store.ListFixturesBySeason(ctx, season.ID)
	if err != nil {
		return SeasonSummary{}, err
	}
	adminLocked, err := s.adminLockedForSeason(season, fixtures)
	if err != nil {
		return SeasonSummary{}, err
	}

	weekCount := 0
	for _, fixture := range fixtures {
		if fixture.WeekNumber > weekCount {
			weekCount = fixture.WeekNumber
		}
	}

	assignedCount := 0
	waitlistCount := 0
	for _, player := range players {
		if player.DivisionID != nil && player.Status == PlayerStatusAssigned {
			assignedCount++
			continue
		}
		waitlistCount++
	}

	return SeasonSummary{
		ID:               season.ID,
		Name:             season.Name,
		Status:           season.Status,
		Timezone:         season.Timezone,
		StartedAt:        season.StartedAt,
		RegistrationOpen: season.RegistrationOpen(),
		SeasonStarted:    !season.RegistrationOpen(),
		AdminLocked:      adminLocked,
		CanStartSeason:   season.RegistrationOpen(),
		CanEditSettings:  !adminLocked,
		CanEditDivisionChannel: !adminLocked,
		CanEditDivisions: !adminLocked,
		CanAssignPlayers: !adminLocked,
		PlayerCount:      len(players),
		WeekCount:        weekCount,
		GameVariant:      season.GameVariant,
		LegsToWin:        season.LegsToWin,
		GamesPerWeek:     season.GamesPerWeek,
		TotalFixtures:    len(fixtures),
		DivisionCount:    len(divisions),
		AssignedCount:    assignedCount,
		WaitlistCount:    waitlistCount,
	}, nil
}

func (s SeasonService) adminLockedForSeason(season Season, fixtures []Fixture) (bool, error) {
	if season.RegistrationOpen() {
		return false, nil
	}
	if len(fixtures) == 0 {
		return false, nil
	}
	loc, err := time.LoadLocation(season.Timezone)
	if err != nil {
		return false, err
	}
	firstReveal := firstFixtureReveal(fixtures, loc)
	if firstReveal == nil {
		return false, nil
	}
	return !s.now().In(loc).Before(*firstReveal), nil
}

func firstFixtureReveal(fixtures []Fixture, loc *time.Location) *time.Time {
	var firstReveal *time.Time
	for _, fixture := range fixtures {
		revealAt := time.Date(fixture.ScheduledAt.In(loc).Year(), fixture.ScheduledAt.In(loc).Month(), fixture.ScheduledAt.In(loc).Day(), 9, 0, 0, 0, loc)
		if firstReveal == nil || revealAt.Before(*firstReveal) {
			value := revealAt
			firstReveal = &value
		}
	}
	return firstReveal
}

func (s SeasonService) StartSeason(ctx context.Context) (SeasonSummary, error) {
	season, players, err := activeSeasonAndPlayers(ctx, s.store)
	if err != nil {
		return SeasonSummary{}, err
	}
	divisions, err := s.store.ListDivisionsBySeason(ctx, season.ID)
	if err != nil {
		return SeasonSummary{}, err
	}

	if !season.RegistrationOpen() {
		return SeasonSummary{}, ErrSeasonAlreadyStarted
	}

	// Validate config at start time.
	if err := ValidateGameVariant(season.GameVariant); err != nil {
		return SeasonSummary{}, err
	}
	if err := ValidateLegsToWin(season.LegsToWin); err != nil {
		return SeasonSummary{}, err
	}
	playersByDivision := make(map[int64][]Player, len(divisions))
	for _, player := range players {
		if player.DivisionID == nil || player.Status != PlayerStatusAssigned {
			continue
		}
		playersByDivision[*player.DivisionID] = append(playersByDivision[*player.DivisionID], player)
	}
	hasEligibleDivision := false
	for _, division := range divisions {
		divisionPlayers := playersByDivision[division.ID]
		if len(divisionPlayers) < 2 {
			continue
		}
		hasEligibleDivision = true
		if err := ValidateGamesPerWeek(season.GamesPerWeek, len(divisionPlayers)); err != nil {
			return SeasonSummary{}, err
		}
	}
	if !hasEligibleDivision {
		return SeasonSummary{}, ErrNotEnoughPlayers
	}

	startedSeason := season.Start(s.now().UTC())
	fixtures := make([]Fixture, 0)
	for _, division := range divisions {
		divisionPlayers := playersByDivision[division.ID]
		if len(divisionPlayers) < 2 {
			continue
		}
		divisionFixtures, err := GenerateRoundRobinFixtures(startedSeason, division, divisionPlayers)
		if err != nil {
			return SeasonSummary{}, err
		}
		fixtures = append(fixtures, divisionFixtures...)
	}

	startedSeason, err = s.store.UpsertSeason(ctx, startedSeason)
	if err != nil {
		return SeasonSummary{}, err
	}

	if _, err := s.store.CreateFixtures(ctx, fixtures); err != nil {
		_, revertErr := s.store.UpsertSeason(ctx, season)
		if revertErr != nil {
			return SeasonSummary{}, errors.Join(err, revertErr)
		}
		return SeasonSummary{}, err
	}

	return s.Summary(ctx)
}

func (s SeasonService) UpdateName(ctx context.Context, name string) (SeasonSummary, error) {
	season, err := s.store.GetActiveSeason(ctx)
	if err != nil {
		return SeasonSummary{}, err
	}

	fixtures, err := s.store.ListFixturesBySeason(ctx, season.ID)
	if err != nil {
		return SeasonSummary{}, err
	}
	adminLocked, err := s.adminLockedForSeason(season, fixtures)
	if err != nil {
		return SeasonSummary{}, err
	}
	if adminLocked {
		return SeasonSummary{}, ErrSeasonRenameLocked
	}

	if err := ValidateSeasonName(name); err != nil {
		return SeasonSummary{}, err
	}

	season.Name = NormalizeSeasonName(name)
	if _, err := s.store.UpsertSeason(ctx, season); err != nil {
		return SeasonSummary{}, err
	}

	return s.Summary(ctx)
}

func (s SeasonService) UpdateConfig(ctx context.Context, gameVariant string, legsToWin, gamesPerWeek int) (SeasonSummary, error) {
	season, err := s.store.GetActiveSeason(ctx)
	if err != nil {
		return SeasonSummary{}, err
	}

	fixtures, err := s.store.ListFixturesBySeason(ctx, season.ID)
	if err != nil {
		return SeasonSummary{}, err
	}
	adminLocked, err := s.adminLockedForSeason(season, fixtures)
	if err != nil {
		return SeasonSummary{}, err
	}
	if adminLocked {
		return SeasonSummary{}, ErrSeasonConfigLocked
	}

	if err := ValidateGameVariant(gameVariant); err != nil {
		return SeasonSummary{}, err
	}
	if err := ValidateLegsToWin(legsToWin); err != nil {
		return SeasonSummary{}, err
	}

	if gamesPerWeek < 1 {
		return SeasonSummary{}, ErrInvalidGamesPerWeek
	}

	season.GameVariant = gameVariant
	season.LegsToWin = legsToWin
	season.GamesPerWeek = gamesPerWeek

	if _, err := s.store.UpsertSeason(ctx, season); err != nil {
		return SeasonSummary{}, err
	}

	return s.Summary(ctx)
}

func (s SeasonService) SchedulePreview(ctx context.Context) (SchedulePreview, error) {
	season, err := s.store.GetActiveSeason(ctx)
	if err != nil {
		return SchedulePreview{}, err
	}
	players, err := s.store.ListPlayersBySeason(ctx, season.ID)
	if err != nil {
		return SchedulePreview{}, err
	}
	divisions, err := s.store.ListDivisionsBySeason(ctx, season.ID)
	if err != nil {
		return SchedulePreview{}, err
	}
	playersByDivision := make(map[int64]int, len(divisions))
	assignedCount := 0
	totalFixtures := 0
	weekCount := 0
	for _, player := range players {
		if player.DivisionID == nil || player.Status != PlayerStatusAssigned {
			continue
		}
		assignedCount++
		playersByDivision[*player.DivisionID]++
	}
	for _, division := range divisions {
		preview := ComputeSchedulePreview(playersByDivision[division.ID], season.GameVariant, season.LegsToWin, season.GamesPerWeek)
		totalFixtures += preview.TotalFixtures
		if preview.WeekCount > weekCount {
			weekCount = preview.WeekCount
		}
	}
	return SchedulePreview{
		PlayerCount:   assignedCount,
		GameVariant:   season.GameVariant,
		LegsToWin:     season.LegsToWin,
		GamesPerWeek:  season.GamesPerWeek,
		WeekCount:     weekCount,
		TotalFixtures: totalFixtures,
	}, nil
}

func (s SeasonService) ListDivisions(ctx context.Context) ([]Division, error) {
	season, err := s.store.GetActiveSeason(ctx)
	if err != nil {
		return nil, err
	}
	return s.store.ListDivisionsBySeason(ctx, season.ID)
}

func (s SeasonService) ProvisionDivisions(ctx context.Context, count int) ([]Division, error) {
	season, err := s.store.GetActiveSeason(ctx)
	if err != nil {
		return nil, err
	}
	fixtures, err := s.store.ListFixturesBySeason(ctx, season.ID)
	if err != nil {
		return nil, err
	}
	adminLocked, err := s.adminLockedForSeason(season, fixtures)
	if err != nil {
		return nil, err
	}
	if adminLocked {
		return nil, ErrDivisionSlugLocked
	}
	if err := ValidateDivisionCount(count); err != nil {
		return nil, err
	}
	divisions := make([]Division, 0, count)
	for i := 1; i <= count; i++ {
		name := "Division " + strconv.Itoa(i)
		divisions = append(divisions, Division{
			SeasonID: season.ID,
			Name:     name,
			Slug:     NormalizeDivisionSlug(name),
			Position: i,
		})
	}
	return s.store.ReplaceDivisions(ctx, season.ID, divisions)
}

func (s SeasonService) UpdateDivision(ctx context.Context, divisionID int64, name, slug, slackPublicChannelID string) (Division, error) {
	season, err := s.store.GetActiveSeason(ctx)
	if err != nil {
		return Division{}, err
	}
	fixtures, err := s.store.ListFixturesBySeason(ctx, season.ID)
	if err != nil {
		return Division{}, err
	}
	adminLocked, err := s.adminLockedForSeason(season, fixtures)
	if err != nil {
		return Division{}, err
	}
	if adminLocked {
		return Division{}, ErrDivisionChannelLocked
	}
	if err := ValidateDivisionName(name); err != nil {
		return Division{}, err
	}
	if err := ValidateDivisionSlug(slug); err != nil {
		return Division{}, err
	}
	divisions, err := s.store.ListDivisionsBySeason(ctx, season.ID)
	if err != nil {
		return Division{}, err
	}
	for _, division := range divisions {
		if division.ID != divisionID && NormalizeDivisionSlug(division.Slug) == NormalizeDivisionSlug(slug) {
			return Division{}, ErrDuplicateDivisionSlug
		}
	}
	for _, division := range divisions {
		if division.ID == divisionID {
			if !adminLocked {
				division.Name = NormalizeDivisionName(name)
			}
			if season.RegistrationOpen() {
				division.Slug = NormalizeDivisionSlug(slug)
			}
			division.SlackPublicChannelID = normalizeSpacing(slackPublicChannelID)
			return s.store.UpdateDivision(ctx, division)
		}
	}
	return Division{}, ErrDivisionNotFound
}

type PublicFixtureWeek struct {
	WeekNumber int
	Status     string
	RevealAt   time.Time
	Fixtures   []PublicFixture
}

type PublicFixture struct {
	ID          int64
	PlayerOne   string
	PlayerTwo   string
	ScheduledAt *time.Time
	GameVariant string
	LegsToWin   int
	Result      *ResultSnapshot
}

type FixtureService struct {
	store Store
	now   func() time.Time
}

func NewFixtureService(store Store) FixtureService {
	return NewFixtureServiceWithNow(store, time.Now)
}

func NewFixtureServiceWithNow(store Store, now func() time.Time) FixtureService {
	return FixtureService{store: store, now: now}
}

func (s FixtureService) PublicSchedule(ctx context.Context, divisionSlug string) ([]PublicFixtureWeek, int, error) {
	season, err := s.store.GetActiveSeason(ctx)
	if err != nil {
		return nil, 0, err
	}
	division, err := s.store.GetDivisionBySlug(ctx, season.ID, divisionSlug)
	if err != nil {
		return nil, 0, err
	}

	fixtures, err := s.store.ListFixturesByDivision(ctx, division.ID)
	if err != nil {
		return nil, 0, err
	}
	if len(fixtures) == 0 {
		return nil, 0, nil
	}

	players, err := s.store.ListPlayersBySeason(ctx, season.ID)
	if err != nil {
		return nil, 0, err
	}
	playersByID := make(map[int64]Player, len(players))
	for _, player := range players {
		playersByID[player.ID] = player
	}
	results, err := s.store.ListResultsByDivision(ctx, division.ID)
	if err != nil {
		return nil, 0, err
	}
	resultsByFixtureID := make(map[int64]Result, len(results))
	for _, result := range results {
		resultsByFixtureID[result.FixtureID] = result
	}

	loc, err := time.LoadLocation(season.Timezone)
	if err != nil {
		return nil, 0, err
	}

	currentWeek := CurrentPublicWeek(fixtures, s.now(), loc)
	grouped := GroupFixturesByWeek(fixtures, loc)
	response := make([]PublicFixtureWeek, 0, len(grouped))
	for _, week := range grouped {
		status := "locked"
		if week.WeekNumber <= currentWeek {
			status = "unlocked"
		}

		publicFixtures := make([]PublicFixture, 0, len(week.Fixtures))
		for index, fixture := range week.Fixtures {
			item := PublicFixture{
				ID:        fixture.ID,
				PlayerOne: playersByID[fixture.PlayerOneID].FixtureLabel(),
				PlayerTwo: playersByID[fixture.PlayerTwoID].FixtureLabel(),
			}
			if status == "unlocked" {
				scheduled := fixture.ScheduledAt
				item.ScheduledAt = &scheduled
				item.GameVariant = fixture.GameVariant
				item.LegsToWin = fixture.LegsToWin
				item.Result = SnapshotFromOptionalResult(resultsByFixtureID[fixture.ID])
			} else {
				placeholders := lockedFixturePlaceholders[index%len(lockedFixturePlaceholders)]
				item.PlayerOne = placeholders[0]
				item.PlayerTwo = placeholders[1]
			}
			publicFixtures = append(publicFixtures, item)
		}

		response = append(response, PublicFixtureWeek{
			WeekNumber: week.WeekNumber,
			Status:     status,
			RevealAt:   week.RevealAt,
			Fixtures:   publicFixtures,
		})
	}

	return response, currentWeek, nil
}

type AdminFixtureWeek struct {
	WeekNumber int
	RevealAt   time.Time
	Fixtures   []AdminFixture
}

type AdminFixture struct {
	ID          int64
	PlayerOne   string
	PlayerTwo   string
	ScheduledAt time.Time
	GameVariant string
	LegsToWin   int
	Status      string
	Result      *ResultSnapshot
}

func (s FixtureService) AdminSchedule(ctx context.Context, divisionSlug string) ([]AdminFixtureWeek, error) {
	season, err := s.store.GetActiveSeason(ctx)
	if err != nil {
		return nil, err
	}
	division, err := s.store.GetDivisionBySlug(ctx, season.ID, divisionSlug)
	if err != nil {
		return nil, err
	}
	fixtures, err := s.store.ListFixturesByDivision(ctx, division.ID)
	if err != nil {
		return nil, err
	}
	if len(fixtures) == 0 {
		return nil, nil
	}
	players, err := s.store.ListPlayersBySeason(ctx, season.ID)
	if err != nil {
		return nil, err
	}
	results, err := s.store.ListResultsByDivision(ctx, division.ID)
	if err != nil {
		return nil, err
	}
	playersByID := make(map[int64]Player, len(players))
	for _, player := range players {
		playersByID[player.ID] = player
	}
	resultsByFixtureID := make(map[int64]Result, len(results))
	for _, result := range results {
		resultsByFixtureID[result.FixtureID] = result
	}
	loc, err := time.LoadLocation(season.Timezone)
	if err != nil {
		return nil, err
	}
	grouped := GroupFixturesByWeek(fixtures, loc)
	response := make([]AdminFixtureWeek, 0, len(grouped))
	for _, week := range grouped {
		adminFixtures := make([]AdminFixture, 0, len(week.Fixtures))
		for _, fixture := range week.Fixtures {
			adminFixtures = append(adminFixtures, AdminFixture{
				ID:          fixture.ID,
				PlayerOne:   playersByID[fixture.PlayerOneID].FixtureLabel(),
				PlayerTwo:   playersByID[fixture.PlayerTwoID].FixtureLabel(),
				ScheduledAt: fixture.ScheduledAt,
				GameVariant: fixture.GameVariant,
				LegsToWin:   fixture.LegsToWin,
				Status:      fixture.Status,
				Result:      SnapshotFromOptionalResult(resultsByFixtureID[fixture.ID]),
			})
		}
		response = append(response, AdminFixtureWeek{WeekNumber: week.WeekNumber, RevealAt: week.RevealAt, Fixtures: adminFixtures})
	}
	return response, nil
}

func SnapshotFromOptionalResult(result Result) *ResultSnapshot {
	if result.ID == 0 {
		return nil
	}
	return SnapshotFromResult(result)
}

type ResultService struct {
	store Store
	now   func() time.Time
}

func NewResultService(store Store) ResultService {
	return NewResultServiceWithNow(store, time.Now)
}

func NewResultServiceWithNow(store Store, now func() time.Time) ResultService {
	return ResultService{store: store, now: now}
}

func (s ResultService) RecordResult(ctx context.Context, fixtureID int64, playerOneLegs, playerTwoLegs int, playerOneAverage, playerTwoAverage *float64) (Result, error) {
	fixture, err := s.store.GetFixture(ctx, fixtureID)
	if err != nil {
		return Result{}, err
	}
	winnerID, err := WinnerIDForFixture(fixture, playerOneLegs, playerTwoLegs)
	if err != nil {
		return Result{}, err
	}

	results, err := s.store.ListResultsByDivision(ctx, fixture.DivisionID)
	if err != nil {
		return Result{}, err
	}
	for _, existing := range results {
		if existing.FixtureID == fixtureID {
			return Result{}, ErrResultAlreadyExists
		}
	}

	now := s.now().UTC()
	result := Result{
		FixtureID:        fixtureID,
		PlayerOneLegs:    playerOneLegs,
		PlayerTwoLegs:    playerTwoLegs,
		PlayerOneAverage: playerOneAverage,
		PlayerTwoAverage: playerTwoAverage,
		WinnerID:         winnerID,
		EnteredAt:        now,
		UpdatedAt:        now,
	}
	return s.store.CreateResult(ctx, result)
}

func (s ResultService) EditResult(ctx context.Context, fixtureID int64, playerOneLegs, playerTwoLegs int, playerOneAverage, playerTwoAverage *float64, actor string) (Result, error) {
	fixture, err := s.store.GetFixture(ctx, fixtureID)
	if err != nil {
		return Result{}, err
	}
	winnerID, err := WinnerIDForFixture(fixture, playerOneLegs, playerTwoLegs)
	if err != nil {
		return Result{}, err
	}

	existing, err := s.store.GetResultByFixture(ctx, fixtureID)
	if err != nil {
		return Result{}, err
	}

	now := s.now().UTC()
	updated := existing
	updated.PlayerOneLegs = playerOneLegs
	updated.PlayerTwoLegs = playerTwoLegs
	updated.PlayerOneAverage = playerOneAverage
	updated.PlayerTwoAverage = playerTwoAverage
	updated.WinnerID = winnerID
	updated.UpdatedAt = now

	updated, err = s.store.UpdateResult(ctx, updated)
	if err != nil {
		return Result{}, err
	}

	_, err = s.store.CreateAuditLog(ctx, AuditLogEntry{
		FixtureID: fixtureID,
		Action:    "result_edited",
		Actor:     actor,
		OldResult: SnapshotFromResult(existing),
		NewResult: SnapshotFromResult(updated),
		CreatedAt: now,
	})
	if err != nil {
		return Result{}, err
	}

	return updated, nil
}

func (s ResultService) DeleteResult(ctx context.Context, fixtureID int64, actor string) error {
	existing, err := s.store.GetResultByFixture(ctx, fixtureID)
	if err != nil {
		return err
	}

	now := s.now().UTC()
	if err := s.store.DeleteResultByFixture(ctx, fixtureID); err != nil {
		return err
	}

	_, err = s.store.CreateAuditLog(ctx, AuditLogEntry{
		FixtureID: fixtureID,
		Action:    "result_deleted",
		Actor:     actor,
		OldResult: SnapshotFromResult(existing),
		NewResult: nil,
		CreatedAt: now,
	})
	return err
}

func (s ResultService) Standings(ctx context.Context, divisionSlug string) ([]StandingRow, error) {
	season, err := s.store.GetActiveSeason(ctx)
	if err != nil {
		return nil, err
	}
	division, err := s.store.GetDivisionBySlug(ctx, season.ID, divisionSlug)
	if err != nil {
		return nil, err
	}
	players, err := s.store.ListPlayersBySeason(ctx, season.ID)
	if err != nil {
		return nil, err
	}
	fixtures, err := s.store.ListFixturesByDivision(ctx, division.ID)
	if err != nil {
		return nil, err
	}
	results, err := s.store.ListResultsByDivision(ctx, division.ID)
	if err != nil {
		return nil, err
	}
	divisionPlayers := make([]Player, 0)
	for _, player := range players {
		if player.DivisionID != nil && *player.DivisionID == division.ID && player.Status == PlayerStatusAssigned {
			divisionPlayers = append(divisionPlayers, player)
		}
	}
	return BuildStandings(divisionPlayers, fixtures, results), nil
}

func (s ResultService) AuditLog(ctx context.Context, divisionSlug string) ([]AuditLogEntry, error) {
	season, err := s.store.GetActiveSeason(ctx)
	if err != nil {
		return nil, err
	}
	division, err := s.store.GetDivisionBySlug(ctx, season.ID, divisionSlug)
	if err != nil {
		return nil, err
	}
	fixtures, err := s.store.ListFixturesByDivision(ctx, division.ID)
	if err != nil {
		return nil, err
	}
	players, err := s.store.ListPlayersBySeason(ctx, season.ID)
	if err != nil {
		return nil, err
	}
	entries, err := s.store.ListAuditLogsByDivision(ctx, division.ID)
	if err != nil {
		return nil, err
	}
	playersByID := make(map[int64]Player, len(players))
	for _, player := range players {
		playersByID[player.ID] = player
	}
	labelsByFixtureID := make(map[int64]string, len(fixtures))
	for _, fixture := range fixtures {
		labelsByFixtureID[fixture.ID] = playersByID[fixture.PlayerOneID].AdminLabel() + " vs " + playersByID[fixture.PlayerTwoID].AdminLabel()
	}
	for i := range entries {
		entries[i].FixtureLabel = labelsByFixtureID[entries[i].FixtureID]
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.After(entries[j].CreatedAt)
	})
	return entries, nil
}
