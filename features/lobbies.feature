Feature: Lobby Discovery

  Background:
    Given the "signaling" backend is running
    And the "testproxy" backend is running

  Scenario: List empty lobby set
    Given "green" is connected as "1u8fw4aph5ypt" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    When "green" requests lobbies with:
      """json
      {}
      """
    Then "green" should receive 0 lobbies

  Scenario: Don't list lobbies from a different game
    Given "green" creates a network for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And "blue" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "19yrzmetd2bn7" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue,yellow" are joined in a lobby
    When "green" requests lobbies with:
      """json
      {}
      """
    Then "green" should receive 0 lobbies

  Scenario: List lobbies that exist
    Given "green" creates a network for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And "blue" is connected as "h5yzwyizlwao" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"

    When "blue" creates a lobby with these settings:
      """json
      {
        "public": true
      }
      """
    And "blue" receives the network event "lobby" with the argument "19yrzmetd2bn7"

    When "green" requests lobbies with:
      """json
      {}
      """
    Then "green" should have received only these lobbies:
      | code          | playerCount |
      | 19yrzmetd2bn7 | 1           |

  Scenario: Only list public lobbies
    Given "green" creates a network for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And "blue" is connected as "h5yzwyizlwao" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And "yellow" is connected as "19yrzmetd2bn7" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"

    When "blue" creates a lobby with these settings:
      """json
      {
        "public": true
      }
      """
    And "blue" receives the network event "lobby" with the argument "3t3cfgcqup9e"
    And "yellow" creates a lobby with these settings:
      """json
      {
        "public": false
      }
      """
    And "yellow" receives the network event "lobby" with the argument "prb67ouj837u"

    When "green" requests lobbies with:
      """json
      {}
      """
    Then "green" should have received only these lobbies:
      | code         | playerCount | public |
      | 3t3cfgcqup9e | 1           | true   |

  Scenario: Filter on playerCount
    Given "green" is connected as "1u8fw4aph5ypt" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
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

    When "green" requests lobbies with:
      """json
      {
        "playerCount": {
          "$gte": 5
        }
      }
      """
    Then "green" should have received only these lobbies:
      | code          | playerCount | public |
      | 4qva9vyurwbbl | 5           | true   |
      | 5qva9vyurwbbl | 6           | true   |
      | 7qva9vyurwbbl | 8           | true   |
      | 9qva9vyurwbbl | 10          | true   |

  Scenario: Filter on customData
    Given "green" is connected as "1u8fw4aph5ypt" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And these lobbies exist:
      | code          | game                                 | playerCount | custom_data        | public | created_at |
      | 0qva9vyurwbbl | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 1           | {"map": "de_nuke"} | true   | 2020-01-01 |
      | 1qva9vyurwbbl | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 1           | {"map": "de_dust"} | true   | 2020-01-02 |
      | 2qva9vyurwbbl | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 1           | {"map": "de_nuke"} | true   | 2020-01-03 |

    When "green" requests lobbies with:
      """json
      {
        "map": "de_nuke",
        "createdAt": {
          "$gte": "2020-01-02"
        }
      }
      """
    Then "green" should have received only these lobbies:
      | code          |
      | 2qva9vyurwbbl |

  Scenario: List empty lobbies
    Given "green" creates a network for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And "blue" is connected as "h5yzwyizlwao" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"

    When "blue" creates a lobby with these settings:
      """json
      {
        "codeFormat": "short",
        "public": true
      }
      """
    And "blue" receives the network event "lobby" with the argument "HC6Y"

    When "blue" disconnects
    Then "blue" receives the network event "close"

    When "green" requests lobbies with:
      """json
      {}
      """
    Then "green" should have received only these lobbies:
      | code | playerCount |
      | HC6Y | 0           |

  Scenario: Filter created lobbies on customData
    Given "green" creates a network for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And "blue" is connected as "h5yzwyizlwao" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"

    And these lobbies exist:
      | code         | game                                 | playerCount | public | custom_data        |
      | 1qva9vyurwbb | 54fa57d5-b4bd-401d-981d-2c13de99be27 | 9           | true   | {"map": "de_nuke"} |
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
    And "green" receives the network event "lobby" with the argument "19yrzmetd2bn7"

    When "blue" requests lobbies with:
      """json
      {
        "map": "de_nuke"
      }
      """
    Then "blue" should have received only these lobbies:
      | code          |
      | 19yrzmetd2bn7 |
      | 3qva9vyurwbb  |

  Scenario: Sort lobbies with a custom order
    Given "green" is connected as "1u8fw4aph5ypt" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And these lobbies exist:
      | code         | game                                 | playerCount | public | created_at |
      | 1qva9vyurwbb | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 1           | true   | 2020-01-03 |
      | 2qva9vyurwbb | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 3           | true   | 2020-01-02 |
      | 3qva9vyurwbb | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 5           | true   | 2020-01-01 |

    When "green" requests lobbies with:
      | filter | {}                    |
      | sort   | { "playerCount": -1 } |
      | limit  | 2                     |
    Then "green" should have received only these lobbies:
      | code         | playerCount |
      | 3qva9vyurwbb | 5           |
      | 2qva9vyurwbb | 3           |
