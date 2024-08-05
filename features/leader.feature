Feature: Lobbies have a leader that can control the lobby

  Background:
    Given the "signaling" backend is running
    And the "testproxy" backend is running


  Scenario: Other players see the creator as the leader
    Given "blue" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "3t3cfgcqup9e" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue" creates a lobby
    And "blue" receives the network event "lobby" with the argument "prb67ouj837u"

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
          "public": false,
          "maxPlayers": 4,
          "hasPassword": false,
          "customData": null,
          "canUpdateBy": "creator",
          "leader": "h5yzwyizlwao",
          "term": 1
        }
      ]
      """


  Scenario: A new leader is elected when the current leader disconnects
    Given "blue,yellow,green" are joined in a lobby for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue" is the leader of the lobby

    When "blue" disconnects
    Then "green" becomes the leader of the lobby
    And "yellow" receives the network event "leader" with the argument "ka9qy8em4vxr"
    And "green" receives the network event "leader" with the argument "ka9qy8em4vxr"


  Scenario: Joining an empty lobby makes you the leader
    Given "blue" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
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
            "h5yzwyizlwao"
          ],
          "playerCount": 1,
          "creator": "foo",
          "public": true,
          "maxPlayers": 4,
          "hasPassword": false,
          "customData": {
            "map": "de_nuke"
          },
          "canUpdateBy": "creator",
          "leader": "h5yzwyizlwao",
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
    Then "yellow" receives the network event "leader" with the argument "ka9qy8em4vxr"
