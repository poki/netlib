Feature: Players can create and connect a network of players

  Background:
    Given the "signaling" backend is running
    And the "testproxy" backend is running


  Scenario: Connections are reconnected before rtc is disconnected
    Given webrtc is intercepted by the testproxy

    Given "blue" is connected and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

    When "blue" creates a lobby
    And "blue" receives the network event "lobby" with the argument "prb67ouj837u"

    When "yellow" connects to the lobby "prb67ouj837u"
    And "blue" receives the network event "connected" with the argument "[Peer: 3t3cfgcqup9e]"

    When the connection between "yellow" and "blue" is interrupted
    And webrtc is no longer intercepted by the testproxy

    And "blue" boardcasts "Hello, world!" over the reliable channel
    And "yellow" receives the network event "message" with the arguments "[Peer: h5yzwyizlwao]", "reliable" and "Hello, world!"
    And "yellow" has not seen the "reconnecting" event


  Scenario: Connections are reconnected when rtc is disconnected
    Given webrtc is intercepted by the testproxy

    Given "blue" is connected and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

    When "blue" creates a lobby
    And "blue" receives the network event "lobby" with the argument "prb67ouj837u"

    When "yellow" connects to the lobby "prb67ouj837u"
    And "blue" receives the network event "connected" with the argument "[Peer: 3t3cfgcqup9e]"

    When the connection between "yellow" and "blue" is interrupted until the first "disconnected" state

    And "blue" boardcasts "Goodbye, world!" over the reliable channel
    Then "yellow" receives the network event "reconnecting" with the argument "[Peer: h5yzwyizlwao]"
    And "yellow" receives the network event "reconnected" with the argument "[Peer: h5yzwyizlwao]"
    And "yellow" receives the network event "message" with the arguments "[Peer: h5yzwyizlwao]", "reliable" and "Goodbye, world!"


  Scenario: A player reconnects when a websocket has been disconnected
    When "green" creates a network for game "de352868-ee35-474c-b703-510a37f911b2"
    Then "green" receives the network event "ready"
    And "green" has recieved the peer ID "h5yzwyizlwao"

    When "green" disconnected from the signaling server
    Then "green" receives the network event "signalingerror" with the argument "Error: signaling socket closed"
    And "green" receives the network event "signalingreconnected"


  Scenario: Two player get disconnected
    Given webrtc is intercepted by the testproxy

    Given "blue" is connected and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue,yellow" are joined in a lobby

    When the connection between "yellow" and "blue" is interrupted

    Then "yellow" receives the network event "reconnecting" with the argument "[Peer: h5yzwyizlwao]"
    And "yellow" receives the network event "disconnected" with the argument "[Peer: h5yzwyizlwao]"
    And "blue" receives the network event "disconnected" with the argument "[Peer: 3t3cfgcqup9e]"
