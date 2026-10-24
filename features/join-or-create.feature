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
    And these lobbies exist:
      | code          | game                                 | playerCount | created_at           | updated_at           | custom_data                                                                           | public |
      | 2qva9vyurwbbl | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 2           | 2026-01-01T00:00:00Z | 2026-01-02T00:00:00Z | {"code":"different","playerCount":99,"createdAt":null,"updatedAt":null,"latency":999} | true   |
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
      | {"latency":{"$lte":250}}                               |

  Scenario: Join-or-create filters latency from the joining player's location
    Given the "testproxy" backend is running
    And these lobbies exist:
      | code          | game                                 | peers   | public |
      | 0qva9vyurwbbl | 123e4567-e89b-12d3-a456-426614174000 | {peerA} | true   |
    And these peers exist:
      | peer  | game                                 | country | region |
      | peerA | 123e4567-e89b-12d3-a456-426614174000 | US      | US-CA  |
    And "blue" is connected as "1u8fw4aph5ypt" with country,region as "US","US-CA" and ready for game "123e4567-e89b-12d3-a456-426614174000"
    When "blue" joins or creates a lobby with these settings:
      """json
      {
        "filter": {
          "latency": {
            "$lt": 100
          }
        }
      }
      """
    Then "blue" receives the network event "lobby" with the argument "0qva9vyurwbbl"
