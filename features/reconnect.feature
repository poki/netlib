Feature: Players can create and connect a network of players

  Background:
    Given the "signaling" backend is running
    And the "testproxy" backend is running


  Scenario: Connections are reconnected before rtc is disconnected
    Given webrtc is intercepted by the testproxy

    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

    When "blue" creates a lobby
    And "blue" receives the network event "lobby" with the argument "19yrzmetd2bn7"

    When "yellow" connects to the lobby "19yrzmetd2bn7"
    And "blue" receives the network event "connected" with the argument "[Peer: h5yzwyizlwao]"

    When the connection between "yellow" and "blue" is interrupted
    And webrtc is no longer intercepted by the testproxy

    And "blue" boardcasts "Hello, world!" over the reliable channel
    And "yellow" receives the network event "message" with the arguments "[Peer: 1u8fw4aph5ypt]", "reliable" and "Hello, world!"
    And "yellow" has not seen any "reconnecting" event


  Scenario: Connections are reconnected when rtc is disconnected
    Given webrtc is intercepted by the testproxy

    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

    When "blue" creates a lobby
    And "blue" receives the network event "lobby" with the argument "19yrzmetd2bn7"

    When "yellow" connects to the lobby "19yrzmetd2bn7"
    And "blue" receives the network event "connected" with the argument "[Peer: h5yzwyizlwao]"

    When the connection between "yellow" and "blue" is interrupted until the first "disconnected" state

    Then "yellow" receives the network event "reconnecting" with the argument "[Peer: 1u8fw4aph5ypt]"
    And "yellow" receives the network event "reconnected" with the argument "[Peer: 1u8fw4aph5ypt]"
    And "blue" boardcasts "Goodbye, world!" over the reliable channel
    And "yellow" receives the network event "message" with the arguments "[Peer: 1u8fw4aph5ypt]", "reliable" and "Goodbye, world!"


  Scenario: A player reconnects when a websocket has been disconnected
    When "green" creates a network for game "de352868-ee35-474c-b703-510a37f911b2"
    Then "green" receives the network event "ready"
    And "green" has recieved the peer ID "1u8fw4aph5ypt"

    When "green" disconnected from the signaling server
    Then "green" receives the network event "signalingerror" with the argument "[socket-error: signaling socket closed]"
    And "green" receives the network event "signalingreconnected"


  Scenario: Two player get disconnected
    Given webrtc is intercepted by the testproxy

    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue,yellow" are joined in a lobby

    When the connection between "yellow" and "blue" is interrupted

    Then "yellow" receives the network event "reconnecting" with the argument "[Peer: 1u8fw4aph5ypt]"
    And "yellow" receives the network event "disconnected" with the argument "[Peer: 1u8fw4aph5ypt]"
    And "blue" receives the network event "disconnected" with the argument "[Peer: h5yzwyizlwao]"


  Scenario: Reconnect with the signaling server
    When "green" creates a network for game "302ce251-5d37-4274-ab44-31e1eb0c376a"
    Then "green" receives the network event "ready"
    And "green" has recieved the peer ID "1u8fw4aph5ypt"

    When the websocket of "green" is reconnected

    Then "green" receives the network event "signalingreconnected"
    And "green" has recieved the peer ID "1u8fw4aph5ypt"


  Scenario: Reconnect with the signaling server
    Given "green" is connected as "1u8fw4aph5ypt" and ready for game "325a2754-1a6f-4578-b768-196463271229"
    And "blue" is connected as "h5yzwyizlwao" and ready for game "325a2754-1a6f-4578-b768-196463271229"

    When "green" creates a lobby
    Then "green" receives the network event "lobby" with the argument "19yrzmetd2bn7"

    When the websocket of "green" is reconnected
    Then "green" receives the network event "signalingreconnected"
    And "green" has recieved the peer ID "1u8fw4aph5ypt"

    When "blue" connects to the lobby "19yrzmetd2bn7"
    Then "green" receives the network event "connected" with the argument "[Peer: h5yzwyizlwao]"
    And "blue" receives the network event "connected" with the argument "[Peer: 1u8fw4aph5ypt]"

