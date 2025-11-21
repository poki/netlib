Feature: Latency

  Background:
    Given the "signaling" backend is running
    And the "testproxy" backend is running


  Scenario: Lobby listings include the latency to the peer
    Given the next peer's latency vector is set to:
      """
      10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10
      """
    And "green" is connected as "1u8fw4aph5ypt" and ready for game "b6f7fc97-8545-4ffd-b714-7cf339048556"
    And "green" creates a lobby with these settings:
      """json
      {
        "public": true
      }
      """
    And "green" receives the network event "lobby" with the argument "h5yzwyizlwao"
    And the next peer's latency vector is set to:
      """
      20, 20, 20, 20, 20, 20, 20, 20, 20, 20, 20
      """
    And "blue" is connected as "19yrzmetd2bn7" and ready for game "b6f7fc97-8545-4ffd-b714-7cf339048556"

    When "blue" requests lobbies with:
      """json
      {}
      """

    Then "blue" should have received only these lobbies:
      | code         | latency |
      | h5yzwyizlwao | 24      |


  Scenario: Lobby with multiple peers
    Given the next peer's latency vector is set to:
      """
      99, 99, 99, 99, 10, 10, 10, 10, 10, 10, 10
      """
    And "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And the next peer's latency vector is set to:
      """
      10, 10, 10, 99, 99, 99, 99, 10, 10, 10, 10
      """
    And "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue,yellow" are joined in a public lobby
    And the next peer's latency vector is set to:
      """
      10, 10, 10, 10, 10, 10, 99, 99, 99, 99, 10
      """
    And "green" is connected as "3t3cfgcqup9e" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

    When "green" requests lobbies with:
      """json
      {}
      """

    Then "green" should have received only these lobbies:
      | code          | latency |
      | 19yrzmetd2bn7 | 89      |


  Scenario: Sort lobbies by latency
    Given the next peer's latency vector is set to:
      """
      30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30
      """
    And "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue" creates a lobby with these settings:
      """json
      {
        "public": true,
        "customData": {
          "map": "de_nuke"
        }
      }
      """
    And the next peer's latency vector is set to:
      """
      99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99
      """
    And "yellow" is connected as "19yrzmetd2bn7" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" creates a lobby with these settings:
      """json
      {
        "public": true,
        "customData": {
          "map": "de_dust"
        }
      }
      """
    And the next peer's latency vector is set to:
      """
      10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10
      """
    And "green" is connected as "prb67ouj837u" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

    When "green" requests lobbies with:
      | filter | {}               |
      | sort   | { "latency": 1 } |
      | limit  | 1                |

    Then "green" should have received only these lobbies:
      | code         | latency | customData        |
      | h5yzwyizlwao | 34      | {"map":"de_nuke"} |


  Scenario: Latency to your own lobby
    Given the next peer's latency vector is set to:
      """
      325, 523, 64, 21, 76, 23, 54, 235, 76, 23, 142
      """
    And "green" is connected as "1u8fw4aph5ypt" and ready for game "b6f7fc97-8545-4ffd-b714-7cf339048556"
    And "green" creates a lobby with these settings:
      """json
      {
        "public": true
      }
      """
    And "green" receives the network event "lobby" with the argument "h5yzwyizlwao"

    When "green" requests lobbies with:
      """json
      {}
      """

    Then "green" should have received only these lobbies:
      | code         | latency |
      | h5yzwyizlwao | 0       |


  Scenario: Peers without latency vectors are not included in the estimate
    Given "green" is connected as "1u8fw4aph5ypt" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And these lobbies exist:
      | code          | game                                 | peers              | public |
      | 1u8fw4aph5ypt | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | {"peer1"}          | true   |
      | 0qva9vyurwbbl | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | {"peer2", "peer3"} | true   |
    And these peers exist:
      | peer  | game                                 | latency_vector                     |
      | peer1 | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | null                               |
      | peer2 | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | null                               |
      | peer3 | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | {10,10,10,10,10,10,10,10,10,10,10} |

    When "green" requests lobbies with:
      | filter | {}               |
      | sort   | { "latency": 1 } |
    Then "green" should have received only these lobbies:
      | code          | latency   |
      | 0qva9vyurwbbl | 10        |
      | 1u8fw4aph5ypt | undefined |


  Scenario: Client without latency vectors gives null latency estimates
    Given the next peer's latency vector is set to:
      """
      null
      """
    Given "green" is connected as "1u8fw4aph5ypt" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And these lobbies exist:
      | code          | game                                 | peers              | public |
      | 0qva9vyurwbbl | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | {"peer1", "peer2"} | true   |
    And these peers exist:
      | peer  | game                                 | latency_vector                     |
      | peer1 | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | {10,10,10,10,10,10,10,10,10,10,10} |

    When "green" requests lobbies with:
      """json
      {}
      """
    Then "green" should have received only these lobbies:
      | code          | latency   |
      | 0qva9vyurwbbl | undefined |
