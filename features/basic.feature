Feature: Players can create and connect a network of players

  Background:
    Given the "signaling" backend is running


  Scenario: A player can create a network to join a game
    When "green" creates a network for game "164aae2e-c6e5-4073-80bf-b2a03ad4c9b7"
    Then "green" receives the network event "ready"
    And "green" has recieved the peer ID "1u8fw4aph5ypt"


  Scenario: A player can create a lobby
    Given "green" is connected as "1u8fw4aph5ypt" and ready for game "b6f7fc97-8545-4ffd-b714-7cf339048556"
    When "green" creates a lobby
    And "green" receives the network event "lobby" with the argument "h5yzwyizlwao"


  Scenario: A player can create a lobby with a short code
    Given "green" is connected as "1u8fw4aph5ypt" and ready for game "b6f7fc97-8545-4ffd-b714-7cf339048556"
    When "green" creates a lobby with these settings:
      """
      {
        "codeFormat": "short"
      }
      """
    And "green" receives the network event "lobby" with the argument "SHC6"


  Scenario: Connect two players to a lobby
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

    When "blue" creates a lobby
    And "blue" receives the network event "lobby" with the argument "19yrzmetd2bn7"

    When "yellow" connects to the lobby "19yrzmetd2bn7"
    And "blue" receives the network event "connected" with the argument "[Peer: h5yzwyizlwao]"
    And "yellow" receives the network event "connected" with the argument "[Peer: 1u8fw4aph5ypt]"


  Scenario: Connect three players to a lobby and broadcast a message
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "green" is connected as "19yrzmetd2bn7" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

    Given "blue,yellow" are joined in a lobby
    When "green" connects to the lobby "3t3cfgcqup9e"
    And "blue" receives the network event "connected" with the argument "[Peer: 19yrzmetd2bn7]"
    And "yellow" receives the network event "connected" with the argument "[Peer: 19yrzmetd2bn7]"
    And "green" receives the network event "connected" with the argument "[Peer: 1u8fw4aph5ypt]"
    And "green" receives the network event "connected" with the argument "[Peer: h5yzwyizlwao]"

    When "blue" boardcasts "Hello, world!" over the reliable channel
    Then "yellow" receives the network event "message" with the arguments "[Peer: 19yrzmetd2bn7]", "reliable" and "Hello, world!"
    And "green" receives the network event "message" with the arguments "[Peer: 19yrzmetd2bn7]", "reliable" and "Hello, world!"


  Scenario: A player leaves a lobby
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "green" is connected as "19yrzmetd2bn7" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

    Given "blue,yellow,green" are joined in a lobby
    When "yellow" disconnects
    Then "yellow" receives the network event "close"
    Then "blue" receives the network event "disconnected" with the argument "[Peer: h5yzwyizlwao]"
    Then "green" receives the network event "disconnected" with the argument "[Peer: h5yzwyizlwao]"
