Feature: Player Counts

  Background:
    Given the "signaling" backend is running
    And the "testproxy" backend is running
    And "green" is connected as "1u8fw4aph5ypt" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"

  Scenario: Count yourself before joining a lobby
    Then "green" should see 1 players online and 0 in a lobby

  Scenario: Count private lobby membership and leaving
    When "green" creates a lobby
    Then "green" should see 1 players online and 1 in a lobby
    When "green" leaves the lobby
    Then "green" should see 1 players online and 0 in a lobby

  Scenario: Count connected sessions only for this game
    Given these peers exist:
      | peer  | game                                 | disconnected |
      | peer0 | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | false        |
      | peer1 | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | true         |
      | peer2 | 4307bd86-e1df-41b8-b9df-e22afcf084bd | false        |
    And these lobbies exist:
      | code    | game                                 | playerCount | public |
      | private | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 2           | false  |
      | public  | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 2           | true   |
      | other   | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 3           | true   |
    Then "green" should see 2 players online and 1 in a lobby

  Scenario: Reject requests after closing
    When "green" disconnects
    And "green" tries to request player counts
    Then the latest error for "green" is "network is closing or not connected"

  Scenario: Stop counting a player who closes their network
    Given "blue" is connected as "h5yzwyizlwao" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And "green,blue" are joined in a lobby
    Then "green" should see 2 players online and 2 in a lobby
    When "blue" disconnects
    Then "green" receives the network event "disconnected"
    And "green" should see 1 players online and 1 in a lobby
