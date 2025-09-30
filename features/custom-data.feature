Feature: customData on lobbies can be used for filtering and extra information

  Background:
    Given the "signaling" backend is running


  Scenario: Connect to a lobby with custom data
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

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
        "19yrzmetd2bn7",
        {
          "code": "19yrzmetd2bn7",
          "peers": [
            "1u8fw4aph5ypt"
          ],
          "playerCount": 1,
          "creator": "1u8fw4aph5ypt",
          "public": true,
          "maxPlayers": 4,
          "hasPassword": false,
          "customData": {
            "gameMode": "deathmatch",
            "map": "de_dust2"
          },
          "canUpdateBy": "creator",
          "leader": "1u8fw4aph5ypt",
          "term": 1
        }
      ]
      """

    When "yellow" connects to the lobby "19yrzmetd2bn7"
    Then "yellow" receives the network event "lobby" with the arguments:
      """json
      [
        "19yrzmetd2bn7",
        {
          "code": "19yrzmetd2bn7",
          "peers": [
            "1u8fw4aph5ypt",
            "h5yzwyizlwao"
          ],
          "playerCount": 2,
          "creator": "1u8fw4aph5ypt",
          "public": true,
          "maxPlayers": 4,
          "hasPassword": false,
          "customData": {
            "gameMode": "deathmatch",
            "map": "de_dust2"
          },
          "canUpdateBy": "creator",
          "leader": "1u8fw4aph5ypt",
          "term": 1
        }
      ]
      """


  Scenario: The creator can edit a lobby
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" creates a lobby with these settings:
      """json
      {
        "public": true,
        "customData": {
          "status": "open"
        }
      }
      """
    And "yellow" receives the network event "lobby" with the argument "19yrzmetd2bn7"

    When "blue" requests lobbies with:
      """json
      {
        "status": "open"
      }
      """
    Then "blue" should have received only these lobbies:
      | code          |
      | 19yrzmetd2bn7 |

    When "yellow" updates the lobby with these settings:
      """json
      {
        "customData": {
          "status": "started"
        }
      }
      """

    When "blue" requests lobbies with:
      """json
      {
        "status": "open"
      }
      """
    Then "blue" should have received only these lobbies:
      | code |


  Scenario: The creator can set can_update_by
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue" creates a lobby with these settings:
      """json
      {
        "public": true,
        "canUpdateBy": "creator"
      }
      """
    And "blue" receives the network event "lobby" with the argument "h5yzwyizlwao"

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
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue" creates a lobby with these settings:
      """json
      {
        "canUpdateBy": "anyone"
      }
      """
    And "blue" receives the network event "lobby" with the argument "19yrzmetd2bn7"
    And "yellow" connects to the lobby "19yrzmetd2bn7"
    And "yellow" receives the network event "lobby" with the argument "19yrzmetd2bn7"

    When "yellow" updates the lobby with these settings:
      """json
      {
        "customData": {
          "status": "started"
        }
      }
      """
    Then "yellow" receives the network event "lobbyUpdated" with the argument "19yrzmetd2bn7"


  Scenario: The creator can update the lobby when canUpdateBy is 'creator' and they are not the leader
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue" creates a lobby with these settings:
      """json
      {
        "canUpdateBy": "creator"
      }
      """
    And "blue" receives the network event "lobby" with the argument "19yrzmetd2bn7"
    And "yellow" connects to the lobby "19yrzmetd2bn7"
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
    Then "blue" receives the network event "lobbyUpdated" with the argument "19yrzmetd2bn7"
    And "yellow" receives the network event "lobbyUpdated" with the arguments:
      """json
      [
        "19yrzmetd2bn7",
        {
          "code": "19yrzmetd2bn7",
          "peers": [
            "1u8fw4aph5ypt",
            "h5yzwyizlwao"
          ],
          "playerCount": 2,
          "creator": "1u8fw4aph5ypt",
          "public": false,
          "maxPlayers": 4,
          "hasPassword": false,
          "customData": {
            "status": "started"
          },
          "canUpdateBy": "creator",
          "leader": "h5yzwyizlwao",
          "term": 2
        }
      ]
      """


  Scenario: The leader can update the lobby if they are not the creator
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue" creates a lobby with these settings:
      """json
      {
        "canUpdateBy": "leader"
      }
      """
    And "blue" receives the network event "lobby" with the argument "19yrzmetd2bn7"
    And "yellow" connects to the lobby "19yrzmetd2bn7"
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
    Then "yellow" receives the network event "lobbyUpdated" with the argument "19yrzmetd2bn7"
