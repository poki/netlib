Feature: Players can atomically find or create a public lobby

  Background:
    Given the "signaling" backend is running


  Scenario: Simultaneous join-or-create requests connect players in the same lobby
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

    When "blue,yellow" join or create simultaneously with a maximum of 2 players
    Then "blue" receives the network event "lobby" with the argument "19yrzmetd2bn7"
    And "yellow" receives the network event "lobby" with the argument "19yrzmetd2bn7"
    And "blue" receives the network event "connected" with the argument "[Peer: h5yzwyizlwao]"
    And "yellow" receives the network event "connected" with the argument "[Peer: 1u8fw4aph5ypt]"

    When "blue" boardcasts "Hello, world!" over the reliable channel
    Then "yellow" receives the network event "message" with the arguments "[Peer: 1u8fw4aph5ypt]", "reliable" and "Hello, world!"

    When "blue" requests lobbies with:
      """json
      {}
      """
    Then "blue" should have received only these lobbies:
      | code          | playerCount | maxPlayers | public | hasPassword |
      | 19yrzmetd2bn7 | 2           | 2          | true   | false       |


  Scenario: Creation settings apply only when join-or-create creates a lobby
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    When "blue" joins or creates a lobby with these settings:
      """json
      {
        "codeFormat": "short",
        "maxPlayers": 3,
        "customData": {
          "mode": "teams"
        },
        "canUpdateBy": "none"
      }
      """
    Then "blue" receives the network event "lobby" with the argument "SHC6"

    When "yellow" creates a network for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    Then "yellow" receives the network event "ready"
    When "yellow" joins or creates a lobby with these settings:
      """json
      {
        "codeFormat": "default",
        "maxPlayers": 1,
        "customData": {
          "mode": "solo"
        },
        "canUpdateBy": "anyone"
      }
      """
    Then "yellow" receives the network event "lobby" with the argument "SHC6"

    When "blue" requests lobbies with:
      """json
      {}
      """
    Then "blue" should have received only these lobbies:
      | code | playerCount | maxPlayers | public | hasPassword | customData       | canUpdateBy |
      | SHC6 | 2           | 3          | true   | false       | {"mode":"teams"} | none        |


  Scenario: Join-or-create filters existing lobbies by custom data
    Given the "testproxy" backend is running
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And these peers exist:
      | peer  | game                                 |
      | peer0 | 4307bd86-e1df-41b8-b9df-e22afcf084bd |
    And these lobbies exist:
      | code          | game                                 | playerCount | custom_data                | public |
      | 0qva9vyurwbbl | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 1           | {"mode":"solo","level":5}  | true   |
      | 1qva9vyurwbbl | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 1           | {"mode":"teams","level":1} | true   |
      | 2qva9vyurwbbl | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 1           | {"mode":"teams","level":5} | true   |
      | 3qva9vyurwbbl | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 1           | {"mode":"teams","level":5} | false  |
    When "blue" joins or creates a lobby with these settings:
      """json
      {
        "filter": {
          "$or": [
            {
              "mode": "teams"
            },
            {
              "mode": "duo"
            }
          ],
          "level": {
            "$gte": 3
          }
        },
        "customData": {
          "mode": "unused"
        }
      }
      """
    Then "blue" receives the network event "lobby" with the argument "2qva9vyurwbbl"


  Scenario: Join-or-create creates a lobby when no custom data matches
    Given the "testproxy" backend is running
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And these peers exist:
      | peer  | game                                 |
      | peer0 | 4307bd86-e1df-41b8-b9df-e22afcf084bd |
    And these lobbies exist:
      | code          | game                                 | playerCount | custom_data     | public |
      | 0qva9vyurwbbl | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 1           | {"mode":"solo"} | true   |
    When "blue" joins or creates a lobby with these settings:
      """json
      {
        "filter": {
          "mode": "teams"
        },
        "codeFormat": "short",
        "customData": {
          "mode": "teams",
          "level": 5
        }
      }
      """
    Then "blue" receives the network event "lobby" with the argument "SHC6"
    When "blue" requests lobbies with:
      """json
      {
        "mode": "teams"
      }
      """
    Then "blue" should have received only these lobbies:
      | code | playerCount | customData                 |
      | SHC6 | 1           | {"level":5,"mode":"teams"} |


  Scenario Outline: Join-or-create uses the same lobby fields as list
    Given the "testproxy" backend is running
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And these peers exist:
      | peer  | game                                 |
      | peer0 | 4307bd86-e1df-41b8-b9df-e22afcf084bd |
      | peer1 | 4307bd86-e1df-41b8-b9df-e22afcf084bd |
    And these lobbies exist:
      | code          | game                                 | playerCount | created_at           | updated_at           | custom_data                                                             | public |
      | 2qva9vyurwbbl | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 2           | 2026-01-01T00:00:00Z | 2026-01-02T00:00:00Z | {"code":"different","playerCount":99,"createdAt":null,"updatedAt":null} | true   |
    When "blue" joins or creates a lobby with these settings:
      """json
      {
        "filter": <filter>
      }
      """
    Then "blue" receives the network event "lobby" with the argument "2qva9vyurwbbl"

    Examples:
      | filter                                                 |
      | {"code":"2qva9vyurwbbl"}                               |
      | {"playerCount":{"$gte":2,"$lt":3}}                     |
      | {"createdAt":{"$gte":"2026-01-01","$lt":"2026-01-02"}} |
      | {"updatedAt":{"$gte":"2026-01-02","$lt":"2026-01-03"}} |


  Scenario Outline: Join-or-create skips lobbies without a live member
    Given the "testproxy" backend is running
    And "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And these peers exist:
      | peer         | game                                 | disconnected | last_seen            |
      | disconnected | 4307bd86-e1df-41b8-b9df-e22afcf084bd | true         | now                  |
      | expired      | 4307bd86-e1df-41b8-b9df-e22afcf084bd | false        | 2000-01-01T00:00:00Z |
      | wrong-game   | 123e4567-e89b-12d3-a456-426614174000 | false        | now                  |
    And these lobbies exist:
      | code          | game                                 | peers   | max_players | public | custom_data                            |
      | 0qva9vyurwbbl | 4307bd86-e1df-41b8-b9df-e22afcf084bd | <peers> | 2           | true   | {"q":"ttt-q2","kind":"ttt-classic-v2"} |
    When "blue" joins or creates a lobby with these settings:
      """json
      {
        "filter": {
          "q": "ttt-q2",
          "kind": {
            "$in": [
              "ttt-classic-v2"
            ]
          }
        },
        "codeFormat": "short",
        "maxPlayers": 2,
        "customData": {
          "q": "ttt-q2",
          "kind": "ttt-classic-v2"
        }
      }
      """
    Then "blue" receives the network event "lobby" with the argument "SHC6"

    Examples:
      | peers          |
      | {disconnected} |
      | {expired}      |
      | {missing}      |
      | {wrong-game}   |


  Scenario: Join-or-create can reuse an old lobby with a live member
    Given the "testproxy" backend is running
    And "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And these peers exist:
      | peer | game                                 |
      | live | 4307bd86-e1df-41b8-b9df-e22afcf084bd |
    And these lobbies exist:
      | code          | game                                 | peers  | max_players | created_at           | public | custom_data                            |
      | 0qva9vyurwbbl | 4307bd86-e1df-41b8-b9df-e22afcf084bd | {live} | 2           | 2000-01-01T00:00:00Z | true   | {"q":"ttt-q2","kind":"ttt-classic-v2"} |
    When "blue" joins or creates a lobby with these settings:
      """json
      {
        "filter": {
          "q": "ttt-q2",
          "kind": {
            "$in": [
              "ttt-classic-v2"
            ]
          }
        },
        "maxPlayers": 2
      }
      """
    Then "blue" receives the network event "lobby" with the argument "0qva9vyurwbbl"


  Scenario Outline: Regional matchmaking tries the player's region first, then neighbors, and creates a lobby if neither matches
    Given "blue" is connected from country "<country>" subdivision "<subdivision>" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And the "testproxy" backend is running
    And these peers exist:
      | peer  | game                                 |
      | peer0 | 4307bd86-e1df-41b8-b9df-e22afcf084bd |
    And these lobbies exist:
      | code   | game                                 | playerCount | public | matchmaking_region |
      | first  | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 1           | true   | <region1>          |
      | second | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 1           | true   | <region2>          |
    And these lobbies exist:
      | code   | game                                 | playerCount | public |
      | legacy | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 1           | true   |
    When "blue" joins or creates a lobby with these settings:
      """json
      {
        "matchmaking": "<mode>",
        "codeFormat": "short"
      }
      """
    Then "blue" receives the network event "lobby" with the argument "<lobby>"

    Examples:
      | country | subdivision | mode             | region1    | region2        | lobby  |
      | SE      |             | region-neighbors | eu-west    | eu-north       | second |
      | PL      |             | region-neighbors | eu-west    | eu-north       | first  |
      | US      | US-CA       | region-neighbors | eu-west    | eu-north       | SHC6   |
      | US      | CA          | region-neighbors | na-east    | na-central     | second |
      | US      | NY          | region-neighbors | eu-west    | na-west        | second |
      | US      | CA          | region-neighbors | eu-west    | na-east        | second |
      | US      |             | region-neighbors | eu-west    | na-west        | second |
      | CA      |             | region-neighbors | eu-west    | na-east        | second |
      | HK      |             | region-neighbors | cn         | asia-southeast | second |
      | CN      |             | region           | asia-east  | cn             | second |
      | PK      |             | region           | asia-south | pk             | second |
      | PK      |             | region-neighbors | gulf       | eu-west        | second |
      | PK      |             | region-neighbors | asia-south | asia-southeast | second |
      | CO      |             | region-neighbors | sa-south   | na-east        | second |
      | MA      |             | region-neighbors | eu-west    | eu-north       | first  |


  Scenario: A lobby created with default matchmaking keeps its creator's region
    Given "blue" is connected from country "US" subdivision "US-CA" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    When "blue" joins or creates a lobby with these settings:
      """json
      {
        "codeFormat": "short",
        "maxPlayers": 2,
        "customData": {
          "mode": "race"
        }
      }
      """
    Then "blue" receives the network event "lobby" with the argument "SHC6"
    Given "green" is connected from country "CA" subdivision "BC" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    When "green" joins or creates a lobby with these settings:
      """json
      {
        "matchmaking": "region-neighbors",
        "filter": {
          "mode": "race"
        }
      }
      """
    Then "green" receives the network event "lobby" with the argument "SHC6"


  Scenario: Regional fallback skips a full lobby in the player's region and applies the game filter to neighbors
    Given the "testproxy" backend is running
    And "blue" is connected from country "SE" subdivision "" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And these peers exist:
      | peer  | game                                 |
      | peer0 | 4307bd86-e1df-41b8-b9df-e22afcf084bd |
    And these lobbies exist:
      | code      | game                                 | playerCount | public | max_players | matchmaking_region | custom_data     |
      | full      | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 1           | true   | 1           | eu-north           | {"mode":"race"} |
      | different | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 1           | true   | 2           | eu-west            | {"mode":"solo"} |
      | available | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 1           | true   | 2           | eu-east            | {"mode":"race"} |
    When "blue" joins or creates a lobby with these settings:
      """json
      {
        "matchmaking": "region-neighbors",
        "filter": {
          "mode": "race"
        }
      }
      """
    Then "blue" receives the network event "lobby" with the argument "available"


  Scenario Outline: Regional options control the order and scope of lobby searches
    Given the "testproxy" backend is running
    And "blue" is connected from country "<country>" subdivision "" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And these peers exist:
      | peer  | game                                 |
      | peer0 | 4307bd86-e1df-41b8-b9df-e22afcf084bd |
    And these lobbies exist:
      | code     | game                                 | playerCount | public | matchmaking_region | custom_data |
      | distant  | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 1           | true   | na-west            | {"level":3} |
      | neighbor | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 1           | true   | eu-east            | {"level":2} |
      | west     | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 1           | true   | eu-west            | {"level":1} |
    And these lobbies exist:
      | code   | game                                 | playerCount | public | custom_data |
      | legacy | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 1           | true   | {"level":4} |
    When "blue" joins or creates a lobby with these settings:
      """json
      {"filter":{"level":{"$gte":<level>,"$lte":<max>}},<setting>"codeFormat":"short"}
      """
    Then "blue" receives the network event "lobby" with the argument "<expected>"

    Examples:
      | country | setting                               | level | max | expected |
      | NL      |                                       | 1     | 3   | west     |
      | NL      |                                       | 2     | 3   | neighbor |
      | NL      |                                       | 3     | 3   | distant  |
      | NL      |                                       | 4     | 4   | legacy   |
      | NL      | "matchmaking":"region-neighbors-any", | 1     | 3   | west     |
      | NL      | "matchmaking":"region-neighbors-any", | 2     | 3   | neighbor |
      | NL      | "matchmaking":"region",               | 2     | 3   | SHC6     |
      | NL      | "matchmaking":"region-neighbors",     | 3     | 3   | SHC6     |
      | NL      | "matchmaking":"any",                  | 3     | 3   | distant  |
      | NL      | "matchmaking":"any",                  | 4     | 4   | legacy   |
      | NL      | "matchmaking":"region-any",           | 3     | 3   | distant  |
      | NL      | "matchmaking":"region-neighbors-any", | 5     | 5   | SHC6     |
      | XX      | "matchmaking":"region",               | 1     | 1   | west     |
      |         | "matchmaking":"region-neighbors",     | 2     | 3   | neighbor |
      | XX      | "matchmaking":"region",               | 3     | 3   | SHC6     |
