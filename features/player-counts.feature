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
    And "green" leaves the lobby
    And "green" should see 1 players online and 0 in a lobby

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

  Scenario: Stop counting a player who closes their network
    Given "blue" is connected as "h5yzwyizlwao" and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And "green,blue" are joined in a lobby
    Then "green" should see 2 players online and 2 in a lobby
    When "blue" disconnects
    Then "green" receives the network event "disconnected"
    And "green" should see 1 players online and 1 in a lobby

  Scenario: Filter lobby membership while keeping online counts game-wide
    Given these peers exist:
      | peer  | game                                 | disconnected |
      | peer0 | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | false        |
      | peer1 | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | true         |
      | peer2 | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | false        |
      | peer3 | 4307bd86-e1df-41b8-b9df-e22afcf084bd | false        |
    And these lobbies exist:
      | code    | game                                 | playerCount | public | custom_data       |
      | private | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 2           | false  | {"kind":"ranked"} |
      | ranked  | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 2           | true   | {"kind":"ranked"} |
      | casual  | f666036d-d9e1-4d70-b0c3-4a68b24a9884 | 3           | true   | {"kind":"casual"} |
      | other   | 4307bd86-e1df-41b8-b9df-e22afcf084bd | 4           | true   | {"kind":"other"}  |
    Then "green" should see 3 players online and 2 in a lobby
    And "green" should see 3 players online and 2 in a lobby matching '{}'
    And "green" should see 3 players online and 1 in a lobby matching '{"kind":"ranked"}'
    And "green" should see 3 players online and 2 in a lobby matching '{"kind":"casual"}'
    And "green" should see 3 players online and 0 in a lobby matching '{"kind":"other"}'
    And "green" should see 3 players online and 1 in a lobby matching '{"code":"private"}'
    And "green" should see 3 players online and 1 in a lobby matching '{"kind":{"$in":["ranked"]},"playerCount":{"$gte":2}}'

  Scenario: Reject invalid lobby filters
    Then "green" should reject player counts matching '{"kind":{"$invalid":"ranked"}}'
