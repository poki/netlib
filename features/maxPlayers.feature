Feature: Lobbies can have a maximum number of players

  Background:
    Given the "signaling" backend is running


  Scenario: Players can set a maximum number of players for a lobby
    Given "blue" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "3t3cfgcqup9e" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "green" is connected as "ka9qy8em4vxr" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

    And "blue" creates a lobby with these settings:
      """json
      {
        "maxPlayers": 2
      }
      """
    And "blue" receives the network event "lobby" with the argument "dhgp75mn2bll"

    When "yellow" connects to the lobby "dhgp75mn2bll"
    Then "yellow" receives the network event "lobby" with the argument "dhgp75mn2bll"

    When "green" connects to the lobby "dhgp75mn2bll"
    Then the last error is "lobby is full"


  Scenario: You can update the maximum number of players for a lobby
    Given "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" creates a lobby with these settings:
      """json
      {
        "public": true,
        "maxPlayers": 2
      }
      """
    And "yellow" receives the network event "lobby" with the argument "19yrzmetd2bn7"

    When "yellow" requests lobbies with this filter:
      """json
      {
      }
      """
    Then "yellow" should have received only these lobbies:
      | code          | maxPlayers |
      | 19yrzmetd2bn7 | 2       |

    When "yellow" updates the lobby with these settings:
      """json
      {
        "maxPlayers": 4
      }
      """

    When "yellow" requests lobbies with this filter:
      """json
      { 
      }
      """
    Then "yellow" should have received only these lobbies:
      | code          | maxPlayers |
      | 19yrzmetd2bn7 | 4      |
