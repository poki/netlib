Feature: Players can create and connect a network of players

  Background:
    Given the "signaling" backend is running

  Scenario: A player can create a network to join a game
    When "green" creates a network for game "164aae2e-c6e5-4073-80bf-b2a03ad4c9b7"
    Then "green" receives the network event "ready"

  Scenario: A player can create a lobby
    Given "green" is connected and ready for game "b6f7fc97-8545-4ffd-b714-7cf339048556"
    When "green" creates a lobby
    Then "green" has recieved the peer ID "19yrzmetd2bn7"
    And "green" receives the network event "lobby" with the argument "h5yzwyizlwao"
