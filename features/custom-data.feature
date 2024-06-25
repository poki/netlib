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
          "public": true,
          "maxPlayers": 0,
          "customData": {
            "gameMode": "deathmatch",
            "map": "de_dust2"
          },
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
          "public": true,
          "maxPlayers": 0,
          "customData": {
            "gameMode": "deathmatch",
            "map": "de_dust2"
          },
          "leader": "h5yzwyizlwao",
          "term": 1
        }
      ]
      """
