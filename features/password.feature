Feature: Lobbies can be password protected

  Background:
    Given the "signaling" backend is running


  Scenario: Players can create password protected lobbies and join them
    Given "blue" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "3t3cfgcqup9e" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue" creates a lobby with these settings:
      """json
      {
        "password": "foobar"
      }
      """
    And "blue" receives the network event "lobby" with the argument "prb67ouj837u"

    When "yellow" connects to the lobby "prb67ouj837u" with the password "foobar"
    Then "yellow" receives the network event "lobby" with the argument "prb67ouj837u"


  Scenario: A wrong password will not allow a player to join a lobby
    Given "blue" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "3t3cfgcqup9e" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue" creates a lobby with these settings:
      """json
      {
        "password": "foobar"
      }
      """
    And "blue" receives the network event "lobby" with the argument "prb67ouj837u"

    When "yellow" tries to connect to the lobby "prb67ouj837u"
    Then the last error for "yellow" is "invalid password"
    And "yellow" is not in a lobby


  Scenario: You can change the password
    Given "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" creates a lobby with these settings:
      """json
      {
        "public": true,
        "password": "foobar"
      }
      """
    And "yellow" receives the network event "lobby" with the argument "19yrzmetd2bn7"

    When "yellow" requests lobbies with this filter:
      """json
      {
      }
      """
    Then "yellow" should have received only these lobbies:
      | code          | hasPassword |
      | 19yrzmetd2bn7 | true        |

    When "yellow" updates the lobby with these settings:
      """json
      {
        "password": ""
      }
      """

    When "yellow" requests lobbies with this filter:
      """json
      {
      }
      """
    Then "yellow" should have received only these lobbies:
      | code          | hasPassword |
      | 19yrzmetd2bn7 | false       |
