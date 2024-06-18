Feature: Lobby Discovery

  Background:
    Given the "signaling" backend is running
    And the "testproxy" backend is running

  Scenario: List empty lobby set
    Given "green" is connected as "h5yzwyizlwao" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    When "green" requests all lobbies
    Then "green" should receive 0 lobbies

  Scenario: Don't list lobbies from a different game
    Given "green" creates a network for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And "blue" is connected as "3t3cfgcqup9e" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "ka9qy8em4vxr" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue,yellow" are joined in a lobby
    When "green" requests all lobbies
    Then "green" should receive 0 lobbies

  Scenario: List lobbies that exist
    Given "green" creates a network for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And "blue" is connected as "3t3cfgcqup9e" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"

    When "blue" creates a lobby with these settings:
      """json
      {
        "public": true
      }
      """
    And "blue" receives the network event "lobby" with the argument "prb67ouj837u"

    When "green" requests all lobbies
    Then "green" should have received only these lobbies:
      | code         | playerCount |
      | prb67ouj837u | 1           |

  Scenario: Only list public lobbies
    Given "green" creates a network for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And "blue" is connected as "3t3cfgcqup9e" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And "yellow" is connected as "ka9qy8em4vxr" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"

    When "blue" creates a lobby with these settings:
      """json
      {
        "public": true
      }
      """
    And "blue" receives the network event "lobby" with the argument "dhgp75mn2bll"
    And "yellow" creates a lobby with these settings:
      """json
      {
        "public": false
      }
      """
    And "yellow" receives the network event "lobby" with the argument "1qva9vyurwbbl"

    When "green" requests all lobbies
    Then "green" should have received only these lobbies:
      | code         | playerCount | public |
      | dhgp75mn2bll | 1           | true   |

  Scenario: Filter on playerCount
    Given "green" is connected as "h5yzwyizlwao" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And these lobbies exist:
      | code          | game                                 | playerCount | public |
      | 0qva9vyurwbbl | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 1           | true   |
      | 1qva9vyurwbbl | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 2           | false  |
      | 2qva9vyurwbbl | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 3           | true   |
      | 3qva9vyurwbbl | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 4           | true   |
      | 4qva9vyurwbbl | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 5           | true   |
      | 5qva9vyurwbbl | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 6           | true   |
      | 6qva9vyurwbbl | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 7           | false  |
      | 7qva9vyurwbbl | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 8           | true   |
      | 8qva9vyurwbbl | 54fa57d5-b4bd-401d-981d-2c13de99be27 | 9           | true   |
      | 9qva9vyurwbbl | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 10          | true   |

    When "green" requests lobbies with this filter:
      """json
      {
        "playerCount": {"$gte": 5}
      }
      """
    Then "green" should have received only these lobbies:
      | code          | playerCount | public |
      | 4qva9vyurwbbl | 5           | true   |
      | 5qva9vyurwbbl | 6           | true   |
      | 7qva9vyurwbbl | 8           | true   |
      | 9qva9vyurwbbl | 10          | true   |

  Scenario: Filter on customData
    Given "green" is connected as "h5yzwyizlwao" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And these lobbies exist:
      | code          | game                                 | playerCount | custom_data        | public | created_at |
      | 0qva9vyurwbbl | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 1           | {"map": "de_nuke"} | true   | 2020-01-01 |
      | 1qva9vyurwbbl | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 1           | {"map": "de_dust"} | true   | 2020-01-02 | 
      | 2qva9vyurwbbl | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 1           | {"map": "de_nuke"} | true   | 2020-01-03 |

    When "green" requests lobbies with this filter:
      """json
      {
        "map": "de_nuke",
        "createdAt": {"$gte": "2020-01-02"}
      }
      """
    Then "green" should have received only these lobbies:
      | code          |
      | 2qva9vyurwbbl |

  Scenario: List empty lobbies
    Given "green" creates a network for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And "blue" is connected as "3t3cfgcqup9e" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"

    When "blue" creates a lobby with these settings:
      """json
      {
        "codeFormat": "short",
        "public": true
      }
      """
    And "blue" receives the network event "lobby" with the argument "52YS"

    When "blue" disconnects
    Then "blue" receives the network event "close"

    When "green" requests all lobbies
    Then "green" should have received only these lobbies:
      | code | playerCount |
      | 52YS | 0           |

  Scenario: Filter created lobbies on customData
    Given "green" creates a network for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And "blue" is connected as "3t3cfgcqup9e" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"

    And these lobbies exist:
      | code         | game                                 | playerCount | public | custom_data        |
      | 1qva9vyurwbb | 54fa57d5-b4bd-401d-981d-2c13de99be27 | 9           | true   | {"map": "de_nuke"} | # different game
      | 2qva9vyurwbb | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 10          | true   | {"map": "de_dust"} |
      | 3qva9vyurwbb | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 10          | true   | {"map": "de_nuke"} |

    When "green" creates a lobby with these settings:
      """json
      {
        "public": true,
        "customData": {
          "map": "de_nuke"
        }
      }
      """
    And "green" receives the network event "lobby" with the argument "prb67ouj837u"

    When "blue" requests lobbies with this filter:
      """json
      {
        "map": "de_nuke"
      }
      """
    Then "blue" should have received only these lobbies:
      | code         |
      | prb67ouj837u |
      | 3qva9vyurwbb |
