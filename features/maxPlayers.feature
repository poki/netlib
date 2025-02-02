Feature: Lobbies can have a maximum number of players

  Background:
    Given the "signaling" backend is running


  Scenario: Players can set a maximum number of players for a lobby
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "green" is connected as "19yrzmetd2bn7" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

    And "blue" creates a lobby with these settings:
      """json
      {
        "maxPlayers": 2
      }
      """
    And "blue" receives the network event "lobby" with the argument "3t3cfgcqup9e"

    When "yellow" connects to the lobby "3t3cfgcqup9e"
    Then "yellow" receives the network event "lobby" with the argument "3t3cfgcqup9e"

    # The lobby is full
    When "green" tries to connect to the lobby "3t3cfgcqup9e" without a password
    Then "green" failed to join the lobby
    And the latest error for "green" is "lobby is full"

    When "yellow" disconnects
    Then "yellow" receives the network event "close"

    # The lobby is not full anymore
    When "green" connects to the lobby "3t3cfgcqup9e"
    Then "green" receives the network event "lobby" with the argument "3t3cfgcqup9e"


  Scenario: You can update the maximum number of players for a lobby
    Given "yellow" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" creates a lobby with these settings:
      """json
      {
        "public": true
      }
      """
    And "yellow" receives the network event "lobby" with the argument "h5yzwyizlwao"

    When "yellow" requests lobbies with this filter:
      """json
      {}
      """
    Then "yellow" should have received only these lobbies:
      | code         | maxPlayers |
      | h5yzwyizlwao | 4          |

    When "yellow" updates the lobby with these settings:
      """json
      {
        "maxPlayers": 12
      }
      """

    When "yellow" requests lobbies with this filter:
      """json
      {}
      """
    Then "yellow" should have received only these lobbies:
      | code         | maxPlayers |
      | h5yzwyizlwao | 12         |
