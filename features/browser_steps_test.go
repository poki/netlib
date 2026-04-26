package features

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/cucumber/godog"
)

func (w *scenarioWorld) registerSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^the "([^"]*)" backend is running$`, w.backendIsRunning)
	ctx.Step(`^webrtc is intercepted by the testproxy$`, w.webrtcIsInterceptedByTestproxy)
	ctx.Step(`^webrtc is no longer intercepted by the testproxy$`, w.webrtcIsNoLongerInterceptedByTestproxy)
	ctx.Step(`^the connection between "([^"]*)" and "([^"]*)" is interrupted until the first "([^"]*)" state$`, w.connectionBetweenPlayersIsInterruptedUntilState)
	ctx.Step(`^the connection between "([^"]*)" and "([^"]*)" is interrupted$`, w.connectionBetweenPlayersIsInterrupted)

	ctx.Step(`^"([^"]*)" is connected as "([^"]*)" and ready for game "([^"]*)"$`, w.playerIsConnectedAndReadyForGame)
	ctx.Step(`^"([^"]*)" is connected as "([^"]*)" with country,region as "([^"]*)","([^"]*)" and ready for game "([^"]*)"$`, w.playerIsConnectedWithGeoAndReadyForGame)
	ctx.Step(`^"([^"]*)" creates a network for game "([^"]*)"$`, w.playerCreatesNetworkForGame)
	ctx.Step(`^"([^"]*)" creates a lobby$`, w.playerCreatesLobby)
	ctx.Step(`^"([^"]*)" creates a lobby with these settings:$`, w.playerCreatesLobbyWithSettings)
	ctx.Step(`^"([^"]*)" connects to the lobby "([^"]*)"$`, w.playerConnectsToLobby)
	ctx.Step(`^"([^"]*)" connects to the lobby "([^"]*)" with the password "([^"]*)"$`, w.playerConnectsToLobbyWithPassword)
	ctx.Step(`^"([^"]*)" tries to connect to the lobby "([^"]*)" with the password "([^"]*)"$`, w.playerTriesToConnectToLobbyWithPassword)
	ctx.Step(`^"([^"]*)" tries to connect to the lobby "([^"]*)" without a password$`, w.playerTriesToConnectToLobbyWithoutPassword)
	ctx.Step(`^"([^"]*)" boardcasts "([^"]*)" over the reliable channel$`, w.playerBroadcastsOverReliableChannel)
	ctx.Step(`^"([^"]*)" disconnects$`, w.playerDisconnects)
	ctx.Step(`^"([^"]*)" leaves the lobby$`, w.playerLeavesLobby)
	ctx.Step(`^"([^"]*)" requests lobbies with:$`, w.playerRequestsLobbiesWith)
	ctx.Step(`^"([^"]*)" updates the lobby with these settings:$`, w.playerUpdatesLobbyWithSettings)
	ctx.Step(`^"([^"]*)" fails to update the lobby with these settings:$`, w.playerFailsToUpdateLobbyWithSettings)
	ctx.Step(`^"([^"]*)" disconnected from the signaling server$`, w.playerDisconnectedFromSignalingServer)
	ctx.Step(`^the websocket of "([^"]*)" is reconnected$`, w.websocketOfPlayerIsReconnected)

	ctx.Step(`^"([^"]*)" receives the network event "([^"]*)"$`, w.playerReceivesNetworkEvent)
	ctx.Step(`^"([^"]*)" receives the network event "([^"]*)" with the argument "([^"]*)"$`, w.playerReceivesNetworkEventWithArgument)
	ctx.Step(`^"([^"]*)" receives the network event "([^"]*)" with the arguments "([^"]*)", "([^"]*)" and "([^"]*)"$`, w.playerReceivesNetworkEventWithThreeArguments)
	ctx.Step(`^"([^"]*)" receives the network event "([^"]*)" with the arguments:$`, w.playerReceivesNetworkEventWithArguments)
	ctx.Step(`^"([^"]*)" has recieved the peer ID "([^"]*)"$`, w.playerHasReceivedPeerID)
	ctx.Step(`^"([^"]*)" should receive ([0-9]+) lobbies$`, w.playerShouldReceiveLobbyCount)
	ctx.Step(`^"([^"]*)" should have received only these lobbies:$`, w.playerShouldHaveReceivedOnlyTheseLobbies)
	ctx.Step(`^"([^"]*)" has not seen any "([^"]*)" event$`, w.playerHasNotSeenAnyEvent)
	ctx.Step(`^"([^"]*)" has not seen a new "([^"]*)" event$`, w.playerHasNotSeenNewEvent)
	ctx.Step(`^"([^"]*)" is the leader of the lobby$`, w.playerIsLeaderOfLobby)
	ctx.Step(`^"([^"]*)" becomes the leader of the lobby$`, w.playerBecomesLeaderOfLobby)
	ctx.Step(`^the latest error for "([^"]*)" is "([^"]*)"$`, w.latestErrorForPlayerIs)
	ctx.Step(`^"([^"]*)" failed to join the lobby$`, w.playerFailedToJoinLobby)

	ctx.Step(`^"([^"]*)" are joined in a lobby$`, w.playersAreJoinedInLobby)
	ctx.Step(`^"([^"]*)" are joined in a public lobby$`, w.playersAreJoinedInPublicLobby)
	ctx.Step(`^"([^"]*)" are joined in a lobby for game "([^"]*)"$`, w.playersAreJoinedInLobbyForGame)
	ctx.Step(`^these lobbies exist:$`, w.theseLobbiesExist)
	ctx.Step(`^these peers exist:$`, w.thesePeersExist)
}

func (w *scenarioWorld) backendIsRunning(backend string) error {
	return w.startBackend(backend)
}

func (w *scenarioWorld) webrtcIsInterceptedByTestproxy() error {
	w.useTestProxy = true
	return nil
}

func (w *scenarioWorld) webrtcIsNoLongerInterceptedByTestproxy() error {
	for _, player := range w.players {
		if err := player.call("disableTestProxy", nil); err != nil {
			return err
		}
	}
	return nil
}

func (w *scenarioWorld) connectionBetweenPlayersIsInterruptedUntilState(player0Name string, player1Name string, state string) error {
	player0, err := w.player(player0Name)
	if err != nil {
		return err
	}
	player1ID, err := w.playerID(player1Name)
	if err != nil {
		return err
	}
	if w.testproxyURL == "" {
		return fmt.Errorf("testproxy not active")
	}
	if err := player0.call("uninterruptOnConnectionState", nil, player1ID, state, w.testproxyURL); err != nil {
		return err
	}
	return w.connectionBetweenPlayersIsInterrupted(player0Name, player1Name)
}

func (w *scenarioWorld) connectionBetweenPlayersIsInterrupted(player0Name string, player1Name string) error {
	player0ID, err := w.playerID(player0Name)
	if err != nil {
		return err
	}
	player1ID, err := w.playerID(player1Name)
	if err != nil {
		return err
	}
	if w.testproxyURL == "" {
		return fmt.Errorf("testproxy not active")
	}
	if err := httpGetOK(fmt.Sprintf("%s/interrupt?id=%s", w.testproxyURL, player0ID+player1ID)); err != nil {
		return err
	}
	return httpGetOK(fmt.Sprintf("%s/interrupt?id=%s", w.testproxyURL, player1ID+player0ID))
}

func (w *scenarioWorld) playerIsConnectedAndReadyForGame(playerName string, peerID string, gameID string) error {
	return w.playerIsConnected(playerName, peerID, gameID, "", "")
}

func (w *scenarioWorld) playerIsConnectedWithGeoAndReadyForGame(playerName string, peerID string, country string, region string, gameID string) error {
	return w.playerIsConnected(playerName, peerID, gameID, country, region)
}

func (w *scenarioWorld) playerIsConnected(playerName string, peerID string, gameID string, country string, region string) error {
	player, err := w.createPlayer(playerName, gameID, country, region)
	if err != nil {
		return err
	}
	if err := w.waitForEvent(player, "ready", nil, true); err != nil {
		return fmt.Errorf("unable to add player %s to network: %w", playerName, err)
	}
	actualID, err := w.playerID(playerName)
	if err != nil {
		return err
	}
	if actualID != peerID {
		return fmt.Errorf("expected peer ID %s but got %s", peerID, actualID)
	}
	return nil
}

func (w *scenarioWorld) playerCreatesNetworkForGame(playerName string, gameID string) error {
	_, err := w.createPlayer(playerName, gameID, "", "")
	return err
}

func (w *scenarioWorld) createPlayer(playerName string, gameID string, country string, region string) (*browserPlayer, error) {
	if w.signalingURL == "" {
		return nil, fmt.Errorf("signaling backend is not running")
	}

	signalingURL, err := withGeo(w.signalingURL, country, region)
	if err != nil {
		return nil, err
	}

	player, err := w.suitePlayer(playerName)
	if err != nil {
		return nil, err
	}

	options := map[string]any{
		"gameID":       gameID,
		"signalingURL": signalingURL,
	}
	if w.useTestProxy {
		if w.testproxyURL == "" {
			return nil, fmt.Errorf("testproxy not active")
		}
		options["testproxyURL"] = w.testproxyURL
	}
	if err := player.call("createPlayer", nil, options); err != nil {
		_ = player.Close()
		delete(w.players, playerName)
		return nil, err
	}
	return player, nil
}

func (w *scenarioWorld) suitePlayer(playerName string) (*browserPlayer, error) {
	if player, ok := w.players[playerName]; ok {
		return player, nil
	}
	player, err := w.newBrowserPlayer(playerName)
	if err != nil {
		return nil, err
	}
	w.players[playerName] = player
	return player, nil
}

func (w *scenarioWorld) playerCreatesLobby(playerName string) error {
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	return player.call("createLobby", nil)
}

func (w *scenarioWorld) playerCreatesLobbyWithSettings(playerName string) error {
	settings, err := w.currentDocJSON()
	if err != nil {
		return err
	}
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	return player.call("createLobby", nil, settings)
}

func (w *scenarioWorld) playerConnectsToLobby(playerName string, lobbyCode string) error {
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	return player.call("joinLobby", nil, lobbyCode)
}

func (w *scenarioWorld) playerConnectsToLobbyWithPassword(playerName string, lobbyCode string, password string) error {
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	return player.call("joinLobby", nil, lobbyCode, password)
}

func (w *scenarioWorld) playerTriesToConnectToLobbyWithPassword(playerName string, lobbyCode string, password string) error {
	return w.playerTriesToConnectToLobby(playerName, lobbyCode, password)
}

func (w *scenarioWorld) playerTriesToConnectToLobbyWithoutPassword(playerName string, lobbyCode string) error {
	return w.playerTriesToConnectToLobby(playerName, lobbyCode, nil)
}

func (w *scenarioWorld) playerTriesToConnectToLobby(playerName string, lobbyCode string, password any) error {
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	var result struct {
		OK      bool   `json:"ok"`
		Message string `json:"message"`
	}
	if password == nil {
		err = player.call("tryJoinLobby", &result, lobbyCode)
	} else {
		err = player.call("tryJoinLobby", &result, lobbyCode, password)
	}
	if err != nil {
		return err
	}
	if !result.OK {
		w.lastError[playerName] = result.Message
	}
	return nil
}

func (w *scenarioWorld) playerBroadcastsOverReliableChannel(playerName string, message string) error {
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	return player.call("broadcast", nil, "reliable", message)
}

func (w *scenarioWorld) playerDisconnects(playerName string) error {
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	return player.call("closeNetwork", nil)
}

func (w *scenarioWorld) playerLeavesLobby(playerName string) error {
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	return player.call("leaveLobby", nil)
}

func (w *scenarioWorld) playerRequestsLobbiesWith(playerName string) error {
	player, err := w.player(playerName)
	if err != nil {
		return err
	}

	if w.stepArg != nil && w.stepArg.DataTable != nil {
		args := rowsHash(w.stepArg.DataTable)
		var filter any
		var sort any
		var limit any
		if raw := args["filter"]; raw != "" {
			if filter, err = parseJSON(raw); err != nil {
				return err
			}
		}
		if raw := args["sort"]; raw != "" {
			if sort, err = parseJSON(raw); err != nil {
				return err
			}
		}
		if raw := args["limit"]; raw != "" {
			var parsed float64
			if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
				return err
			}
			limit = parsed
		}
		return player.call("listLobbies", nil, filter, sort, limit)
	}

	filter, err := w.currentDocJSON()
	if err != nil {
		return err
	}
	return player.call("listLobbies", nil, filter)
}

func (w *scenarioWorld) playerUpdatesLobbyWithSettings(playerName string) error {
	settings, err := w.currentDocJSON()
	if err != nil {
		return err
	}
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	return player.call("setLobbySettings", nil, settings)
}

func (w *scenarioWorld) playerFailsToUpdateLobbyWithSettings(playerName string) error {
	settings, err := w.currentDocJSON()
	if err != nil {
		return err
	}
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	if err := player.call("setLobbySettings", nil, settings); err == nil {
		return fmt.Errorf("no error thrown")
	}
	return nil
}

func (w *scenarioWorld) playerDisconnectedFromSignalingServer(playerName string) error {
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	return player.call("closeSignalingSocket", nil)
}

func (w *scenarioWorld) websocketOfPlayerIsReconnected(playerName string) error {
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	return player.call("forceReconnectSignaling", nil)
}

func (w *scenarioWorld) playerReceivesNetworkEvent(playerName string, eventName string) error {
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	return w.waitForEvent(player, eventName, nil, true)
}

func (w *scenarioWorld) playerReceivesNetworkEventWithArgument(playerName string, eventName string, expectedArgument string) error {
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	return w.waitForEvent(player, eventName, []any{expectedArgument}, true)
}

func (w *scenarioWorld) playerReceivesNetworkEventWithThreeArguments(playerName string, eventName string, arg0 string, arg1 string, arg2 string) error {
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	return w.waitForEvent(player, eventName, []any{arg0, arg1, arg2}, true)
}

func (w *scenarioWorld) playerReceivesNetworkEventWithArguments(playerName string, eventName string) error {
	args, err := w.currentDocJSON()
	if err != nil {
		return err
	}
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	return w.waitForEvent(player, eventName, args, true)
}

func (w *scenarioWorld) playerHasReceivedPeerID(playerName string, expectedID string) error {
	id, err := w.playerID(playerName)
	if err != nil {
		return err
	}
	if id == "" {
		player, err := w.player(playerName)
		if err != nil {
			return err
		}
		if err := w.waitForEvent(player, "ready", nil, false); err != nil {
			return err
		}
		id, err = w.playerID(playerName)
		if err != nil {
			return err
		}
	}
	if id != expectedID {
		return fmt.Errorf("expected peer ID %s but got %s", expectedID, id)
	}
	return nil
}

func (w *scenarioWorld) playerShouldReceiveLobbyCount(playerName string, expectedLobbyCount int) error {
	lobbies, err := w.playerLobbies(playerName)
	if err != nil {
		return err
	}
	if len(lobbies) != expectedLobbyCount {
		return fmt.Errorf("expected %d lobbies but got %d", expectedLobbyCount, len(lobbies))
	}
	return nil
}

func (w *scenarioWorld) playerShouldHaveReceivedOnlyTheseLobbies(playerName string) error {
	if w.stepArg == nil || w.stepArg.DataTable == nil {
		return fmt.Errorf("expected lobby table")
	}
	lobbies, err := w.playerLobbies(playerName)
	if err != nil {
		return err
	}
	expectedRows := tableHashes(w.stepArg.DataTable)
	headers := tableHeaders(w.stepArg.DataTable)

	for _, row := range expectedRows {
		code := row["code"]
		var matches []map[string]any
		for _, lobby := range lobbies {
			if fmt.Sprint(lobby["code"]) == code {
				matches = append(matches, lobby)
			}
		}
		if len(matches) != 1 {
			return fmt.Errorf("expected to find one lobby with code %s but found %d in [%s]", code, len(matches), lobbyCodes(lobbies))
		}
		lobby := matches[0]
		for _, header := range headers {
			want := row[header]
			got := lobby[header]
			if gotMap, ok := got.(map[string]any); ok {
				if compactJSON(gotMap) != compactExpectedJSON(want) {
					return fmt.Errorf("expected %s to be %s but got %s", header, want, compactJSON(gotMap))
				}
				continue
			}
			if fmt.Sprint(got) != want {
				return fmt.Errorf("expected %s to be %s but got %v", header, want, got)
			}
		}
	}
	if len(lobbies) != len(expectedRows) {
		return fmt.Errorf("expected %d lobbies but got %d", len(expectedRows), len(lobbies))
	}
	return nil
}

func compactExpectedJSON(raw string) string {
	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return raw
	}
	return compactJSON(parsed)
}

func lobbyCodes(lobbies []map[string]any) string {
	codes := make([]string, 0, len(lobbies))
	for _, lobby := range lobbies {
		codes = append(codes, fmt.Sprint(lobby["code"]))
	}
	return strings.Join(codes, ", ")
}

func (w *scenarioWorld) playerHasNotSeenAnyEvent(playerName string, eventName string) error {
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	var event any
	if err := player.call("findEvent", &event, eventName, []any{}); err != nil {
		return err
	}
	if event != nil {
		return fmt.Errorf("%s has recieved a %s event: %s", playerName, eventName, compactJSON(event))
	}
	return nil
}

func (w *scenarioWorld) playerHasNotSeenNewEvent(playerName string, eventName string) error {
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	var event any
	if err := player.call("findNewEvent", &event, eventName, []any{}); err != nil {
		return err
	}
	if event != nil {
		return fmt.Errorf("%s has recieved a %s event: %s", playerName, eventName, compactJSON(event))
	}
	return nil
}

func (w *scenarioWorld) playerIsLeaderOfLobby(playerName string) error {
	id, err := w.playerID(playerName)
	if err != nil {
		return err
	}
	leader, err := w.playerCurrentLeader(playerName)
	if err != nil {
		return err
	}
	if leader != id {
		return fmt.Errorf("player is not the leader")
	}
	return nil
}

func (w *scenarioWorld) playerBecomesLeaderOfLobby(playerName string) error {
	id, err := w.playerID(playerName)
	if err != nil {
		return err
	}
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	if err := w.waitForEvent(player, "leader", []any{id}, false); err != nil {
		return fmt.Errorf("no event leader(%s) received: %w", id, err)
	}
	leader, err := w.playerCurrentLeader(playerName)
	if err != nil {
		return err
	}
	if leader != id {
		return fmt.Errorf("player is not the leader")
	}
	return nil
}

func (w *scenarioWorld) latestErrorForPlayerIs(playerName string, message string) error {
	if got := w.lastError[playerName]; got != message {
		return fmt.Errorf("expected error to be %q but got %q", message, got)
	}
	return nil
}

func (w *scenarioWorld) playerFailedToJoinLobby(playerName string) error {
	player, err := w.player(playerName)
	if err != nil {
		return err
	}
	var lobby string
	if err := player.call("getCurrentLobby", &lobby); err != nil {
		return err
	}
	if lobby != "" {
		return fmt.Errorf("player is in lobby %s", lobby)
	}
	return nil
}

func (w *scenarioWorld) playersAreJoinedInLobby(playerNamesRaw string) error {
	return w.playersAreJoinedInALobby(playerNamesRaw, false)
}

func (w *scenarioWorld) playersAreJoinedInPublicLobby(playerNamesRaw string) error {
	return w.playersAreJoinedInALobby(playerNamesRaw, true)
}

func (w *scenarioWorld) playersAreJoinedInLobbyForGame(playerNamesRaw string, gameID string) error {
	playerNames := splitPlayerNames(playerNamesRaw)
	if len(playerNames) < 2 {
		return fmt.Errorf("need at least 2 players to join a lobby")
	}
	for _, playerName := range playerNames {
		player, err := w.createPlayer(playerName, gameID, "", "")
		if err != nil {
			return err
		}
		if err := w.waitForEvent(player, "ready", nil, true); err != nil {
			return fmt.Errorf("unable to add player %s to network: %w", playerName, err)
		}
	}
	return w.playersAreJoinedInALobby(playerNamesRaw, false)
}

func (w *scenarioWorld) playersAreJoinedInALobby(playerNamesRaw string, public bool) error {
	playerNames := splitPlayerNames(playerNamesRaw)
	if len(playerNames) < 2 {
		return fmt.Errorf("need at least 2 players to join a lobby")
	}
	first, err := w.player(playerNames[0])
	if err != nil {
		return err
	}

	settings := map[string]any{"public": public}
	if err := first.call("createLobby", nil, settings); err != nil {
		return err
	}
	event, err := w.eventFor(first, "lobby", nil, true)
	if err != nil {
		return err
	}
	lobbyCode, _ := event.PayloadString(0)

	for _, playerName := range playerNames[1:] {
		player, err := w.player(playerName)
		if err != nil {
			return err
		}
		if err := player.call("joinLobby", nil, lobbyCode); err != nil {
			return err
		}
		if err := w.waitForEvent(player, "lobby", nil, true); err != nil {
			return err
		}
	}

	for _, playerName := range playerNames {
		player, err := w.player(playerName)
		if err != nil {
			return err
		}
		for i := 0; i < len(playerNames)-1; i++ {
			if err := w.waitForEvent(player, "connected", nil, true); err != nil {
				return err
			}
		}
		count, err := w.playerPeerCount(playerName)
		if err != nil {
			return err
		}
		if count != len(playerNames)-1 {
			return fmt.Errorf("player not connected with enough others")
		}
	}
	return nil
}

func (w *scenarioWorld) theseLobbiesExist() error {
	if w.stepArg == nil || w.stepArg.DataTable == nil {
		return fmt.Errorf("expected lobbies table")
	}
	if w.testproxyURL == "" {
		return fmt.Errorf("testproxy not active")
	}

	headers := tableHeaders(w.stepArg.DataTable)
	rows := tableHashes(w.stepArg.DataTable)
	columns := make([]string, 0, len(headers))
	values := make([]string, 0, len(rows))

	for _, row := range rows {
		rowValues := make([]string, 0, len(headers))
		for _, key := range headers {
			value := row[key]
			column := key
			sqlValue := ""
			if key == "playerCount" {
				column = "peers"
				n := 0
				_, _ = fmt.Sscanf(value, "%d", &n)
				peers := make([]string, 0, n)
				for i := 0; i < n; i++ {
					peers = append(peers, sqlQuote(fmt.Sprintf("peer%d", i)))
				}
				sqlValue = fmt.Sprintf("ARRAY[%s]::VARCHAR(20)[]", strings.Join(peers, ", "))
			} else if value == "null" {
				sqlValue = "NULL"
			} else {
				sqlValue = sqlQuote(value)
			}
			if !contains(columns, column) {
				columns = append(columns, column)
			}
			rowValues = append(rowValues, sqlValue)
		}
		values = append(values, fmt.Sprintf("(%s)", strings.Join(rowValues, ", ")))
	}

	sql := "INSERT INTO lobbies (" + strings.Join(columns, ", ") + ") VALUES " + strings.Join(values, ", ")
	return httpPostOK(w.testproxyURL+"/sql", sql)
}

func (w *scenarioWorld) thesePeersExist() error {
	if w.stepArg == nil || w.stepArg.DataTable == nil {
		return fmt.Errorf("expected peers table")
	}
	if w.testproxyURL == "" {
		return fmt.Errorf("testproxy not active")
	}

	headers := tableHeaders(w.stepArg.DataTable)
	rows := tableHashes(w.stepArg.DataTable)
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		rowValues := make([]string, 0, len(headers))
		for _, key := range headers {
			value := row[key]
			if value == "null" {
				rowValues = append(rowValues, "NULL")
			} else {
				rowValues = append(rowValues, sqlQuote(value))
			}
		}
		values = append(values, fmt.Sprintf("(%s)", strings.Join(rowValues, ", ")))
	}
	sql := "INSERT INTO peers (" + strings.Join(headers, ", ") + ") VALUES " + strings.Join(values, ", ")
	return httpPostOK(w.testproxyURL+"/sql", sql)
}

func (w *scenarioWorld) player(playerName string) (*browserPlayer, error) {
	player, ok := w.players[playerName]
	if !ok {
		return nil, fmt.Errorf("no such player: %s", playerName)
	}
	return player, nil
}

func (w *scenarioWorld) playerID(playerName string) (string, error) {
	player, err := w.player(playerName)
	if err != nil {
		return "", err
	}
	var id string
	if err := player.call("getID", &id); err != nil {
		return "", err
	}
	return id, nil
}

func (w *scenarioWorld) playerPeerCount(playerName string) (int, error) {
	player, err := w.player(playerName)
	if err != nil {
		return 0, err
	}
	var count int
	if err := player.call("getPeerCount", &count); err != nil {
		return 0, err
	}
	return count, nil
}

func (w *scenarioWorld) playerCurrentLeader(playerName string) (string, error) {
	player, err := w.player(playerName)
	if err != nil {
		return "", err
	}
	var leader string
	if err := player.call("getCurrentLeader", &leader); err != nil {
		return "", err
	}
	return leader, nil
}

func (w *scenarioWorld) playerLobbies(playerName string) ([]map[string]any, error) {
	player, err := w.player(playerName)
	if err != nil {
		return nil, err
	}
	var lobbies []map[string]any
	if err := player.call("getLastReceivedLobbies", &lobbies); err != nil {
		return nil, err
	}
	return lobbies, nil
}

func (w *scenarioWorld) waitForEvent(player *browserPlayer, eventName string, matchArguments any, consume bool) error {
	_, err := w.eventFor(player, eventName, matchArguments, consume)
	return err
}

func (w *scenarioWorld) eventFor(player *browserPlayer, eventName string, matchArguments any, consume bool) (recordedEvent, error) {
	if matchArguments == nil {
		matchArguments = []any{}
	}
	var event recordedEvent
	err := player.call("waitForEvent", &event, eventName, matchArguments, consume, int(waitEventTimeout/time.Millisecond))
	return event, err
}

type recordedEvent struct {
	EventName string `json:"eventName"`
	Payload   []any  `json:"eventPayload"`
}

func (e recordedEvent) PayloadString(index int) (string, bool) {
	if index >= len(e.Payload) {
		return "", false
	}
	return fmt.Sprint(e.Payload[index]), true
}

func (w *scenarioWorld) currentDocJSON() (any, error) {
	if w.stepArg == nil || w.stepArg.DocString == nil {
		return nil, fmt.Errorf("expected doc string")
	}
	return parseJSON(w.stepArg.DocString.Content)
}

func splitPlayerNames(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func contains(values []string, needle string) bool {
	return slices.Contains(values, needle)
}
