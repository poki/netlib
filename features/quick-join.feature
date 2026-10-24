Feature: Players can atomically find or create a public lobby

  Background:
    Given the "signaling" backend is running

  Scenario: Simultaneous quick-joins connect players in the same lobby
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

    When "blue,yellow" quick-join simultaneously with a maximum of 2 players
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
      | code         | playerCount | maxPlayers | public | hasPassword |
      | 19yrzmetd2bn7 | 2           | 2          | true   | false       |

  Scenario: Creation settings apply only when quick-join creates a lobby
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    When "blue" quick-joins with these settings:
      """json
      {
        "codeFormat": "short",
        "maxPlayers": 3,
        "customData": { "mode": "teams" },
        "canUpdateBy": "none"
      }
      """
    Then "blue" receives the network event "lobby" with the argument "SHC6"

    When "yellow" creates a network for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    Then "yellow" receives the network event "ready"
    When "yellow" quick-joins with these settings:
      """json
      {
        "codeFormat": "default",
        "maxPlayers": 1,
        "customData": { "mode": "solo" },
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
