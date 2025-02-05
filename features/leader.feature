Feature: Lobbies have a leader that can control the lobby

  Background:
    Given the "signaling" backend is running
    And the "testproxy" backend is running


  Scenario: Other players see the creator as the leader
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue" creates a lobby
    And "blue" receives the network event "lobby" with the argument "19yrzmetd2bn7"

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
          "public": false,
          "maxPlayers": 4,
          "hasPassword": false,
          "customData": null,
          "canUpdateBy": "creator",
          "leader": "1u8fw4aph5ypt",
          "term": 1
        }
      ]
      """


  Scenario: A new leader is elected when the current leader disconnects
    Given "blue,yellow,green" are joined in a lobby for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue" is the leader of the lobby

    When "blue" disconnects
    Then "green" becomes the leader of the lobby
    And "yellow" receives the network event "leader" with the argument "19yrzmetd2bn7"
    And "green" receives the network event "leader" with the argument "19yrzmetd2bn7"


  Scenario: Joining an empty lobby makes you the leader
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And these lobbies exist:
      | code         | game                                 | playerCount | public | custom_data        | creator |
      | 1qva9vyurwbb | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 0           | true   | {"map": "de_nuke"} | foo     |

    When "blue" connects to the lobby "1qva9vyurwbb"
    And "blue" receives the network event "lobby" with the arguments:
      """json
      [
        "1qva9vyurwbb",
        {
          "code": "1qva9vyurwbb",
          "peers": [
            "1u8fw4aph5ypt"
          ],
          "playerCount": 1,
          "creator": "foo",
          "public": true,
          "maxPlayers": 64,
          "hasPassword": false,
          "customData": {
            "map": "de_nuke"
          },
          "canUpdateBy": "creator",
          "leader": "1u8fw4aph5ypt",
          "term": 1
        }
      ]
      """


  Scenario: A player reconnects when a websocket gets a leader event
    Given "blue,yellow,green" are joined in a lobby for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

    When "yellow" disconnected from the signaling server
    And "blue" disconnected from the signaling server
    Then "green" becomes the leader of the lobby

    When "yellow" receives the network event "signalingreconnected"
    Then "yellow" receives the network event "leader" with the argument "19yrzmetd2bn7"
