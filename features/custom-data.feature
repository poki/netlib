Feature: customData on lobbies can be used for filtering and extra information

  Background:
    Given the "signaling" backend is running


  Scenario: Connect to a lobby with custom data
    Given "blue" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "3t3cfgcqup9e" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

    When "blue" creates a lobby with these settings:
      """json
      {
        "public": true,
        "customData": {
          "gameMode": "deathmatch",
          "map": "de_dust2"
        }
      }
      """
    Then "blue" receives the network event "lobby" with the arguments:
      """json
      [
        "prb67ouj837u",
        {
          "code": "prb67ouj837u",
          "peers": [
            "h5yzwyizlwao"
          ],
          "playerCount": 1,
          "creator": "h5yzwyizlwao",
          "public": true,
          "maxPlayers": 4,
          "hasPassword": false,
          "customData": {
            "gameMode": "deathmatch",
            "map": "de_dust2"
          },
          "canUpdateBy": "creator",
          "leader": "h5yzwyizlwao",
          "term": 1
        }
      ]
      """

    When "yellow" connects to the lobby "prb67ouj837u"
    Then "yellow" receives the network event "lobby" with the arguments:
      """json
      [
        "prb67ouj837u",
        {
          "code": "prb67ouj837u",
          "peers": [
            "3t3cfgcqup9e",
            "h5yzwyizlwao"
          ],
          "playerCount": 2,
          "creator": "h5yzwyizlwao",
          "public": true,
          "maxPlayers": 4,
          "hasPassword": false,
          "customData": {
            "gameMode": "deathmatch",
            "map": "de_dust2"
          },
          "canUpdateBy": "creator",
          "leader": "h5yzwyizlwao",
          "term": 1
        }
      ]
      """


  Scenario: The creator can edit a lobby
    Given "blue" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "3t3cfgcqup9e" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" creates a lobby with these settings:
      """json
      {
        "public": true,
        "customData": {
          "status": "open"
        }
      }
      """
    And "yellow" receives the network event "lobby" with the argument "prb67ouj837u"

    When "blue" requests lobbies with this filter:
      """json
      {
        "status": "open"
      }
      """
    Then "blue" should have received only these lobbies:
      | code         |
      | prb67ouj837u |

    When "yellow" updates the lobby with these settings:
      """json
      {
        "customData": {
          "status": "started"
        }
      }
      """

    When "blue" requests lobbies with this filter:
      """json
      {
        "status": "open"
      }
      """
    Then "blue" should have received only these lobbies:
      | code |


  Scenario: The creator can set can_update_by
    Given "blue" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue" creates a lobby with these settings:
      """json
      {
        "public": true,
        "canUpdateBy": "creator"
      }
      """
    And "blue" receives the network event "lobby" with the argument "19yrzmetd2bn7"

    When "blue" updates the lobby with these settings:
      """json
      {
        "canUpdateBy": "none"
      }
      """
    Then "blue" fails to update the lobby with these settings:
      """json
      {
        "customData": {
          "status": "started"
        }
      }
      """


  Scenario: Other players can update the lobby if canUpdateBy is 'anyone'
    Given "blue" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "3t3cfgcqup9e" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue" creates a lobby with these settings:
      """json
      {
        "canUpdateBy": "anyone"
      }
      """
    And "blue" receives the network event "lobby" with the argument "prb67ouj837u"
    And "yellow" connects to the lobby "prb67ouj837u"
    And "yellow" receives the network event "lobby" with the argument "prb67ouj837u"

    When "yellow" updates the lobby with these settings:
      """json
      {
        "customData": {
          "status": "started"
        }
      }
      """
    Then "yellow" receives the network event "lobbyUpdated" with the argument "prb67ouj837u"


  Scenario: The creator can update the lobby when canUpdateBy is 'creator' and they are not the leader
    Given "blue" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "3t3cfgcqup9e" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue" creates a lobby with these settings:
      """json
      {
        "canUpdateBy": "creator"
      }
      """
    And "blue" receives the network event "lobby" with the argument "prb67ouj837u"
    And "yellow" connects to the lobby "prb67ouj837u"
    And "blue" disconnected from the signaling server
    And "yellow" becomes the leader of the lobby
    And "blue" receives the network event "signalingreconnected"

    When "blue" updates the lobby with these settings:
      """json
      {
        "customData": {
          "status": "started"
        }
      }
      """
    Then "blue" receives the network event "lobbyUpdated" with the argument "prb67ouj837u"
    And "yellow" receives the network event "lobbyUpdated" with the arguments:
      """json
      [
        "prb67ouj837u",
        {
          "code": "prb67ouj837u",
          "peers": [
            "3t3cfgcqup9e",
            "h5yzwyizlwao"
          ],
          "playerCount": 2,
          "creator": "h5yzwyizlwao",
          "public": false,
          "maxPlayers": 4,
          "hasPassword": false,
          "customData": {
            "status": "started"
          },
          "canUpdateBy": "creator",
          "leader": "3t3cfgcqup9e",
          "term": 2
        }
      ]
      """


  Scenario: The leader can update the lobby if they are not the creator
    Given "blue" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "3t3cfgcqup9e" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue" creates a lobby with these settings:
      """json
      {
        "canUpdateBy": "leader"
      }
      """
    And "blue" receives the network event "lobby" with the argument "prb67ouj837u"
    And "yellow" connects to the lobby "prb67ouj837u"
    And "blue" disconnects
    And "blue" receives the network event "close"
    And "yellow" becomes the leader of the lobby

    When "yellow" updates the lobby with these settings:
      """json
      {
        "customData": {
          "status": "started"
        }
      }
      """
    Then "yellow" receives the network event "lobbyUpdated" with the argument "prb67ouj837u"
