Feature: Lobby Discovery

  Background:
    Given the "signaling" backend is running

  Scenario: List empty lobby set
    Given "green" is connected and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    When "green" requests all lobbies
    Then "green" should receive 0 lobbies

  Scenario: Don't list lobbies from a different game
    Given "green" creates a network for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And "blue" is connected and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue,yellow" are joined in a lobby
    When "green" requests all lobbies
    Then "green" should receive 0 lobbies

  Scenario: List lobbies that exist
    Given "green" creates a network for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"
    And "blue" is connected and ready for game "f666036d-d9e1-4d70-b0c3-4a68b24a9884"

    When "blue" creates a lobby
    And "blue" receives the network event "lobby" with the argument "prb67ouj837u"

    When "green" requests all lobbies
    Then "green" should have received only these lobbies
      | code         | playerCount |
      | prb67ouj837u | 1           |





