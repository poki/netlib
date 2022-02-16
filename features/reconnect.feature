Feature: Players can create and connect a network of players

  Background:
    Given the "signaling" backend is running
    And the "testproxy" backend is running


  Scenario: Connections are reconnected before rtc is disconnected
    Given webrtc is intercepted by the testproxy

    Given "blue" is connected and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

    When "blue" creates a lobby
    And "blue" receives the network event "lobby" with the argument "19yrzmetd2bn7"

    When "yellow" connects to the lobby "19yrzmetd2bn7"
    And "blue" receives the network event "connected" with the argument "[Peer: prb67ouj837u]"

    When the connection between "yellow" and "blue" is interrupted
    And webrtc is no longer intercepted by the testproxy

    And "blue" boardcasts "Hello, world!" over the reliable channel
    And "yellow" receives the network event "message" with the arguments "[Peer: 3t3cfgcqup9e]", "reliable" and "Hello, world!"
    And "yellow" has not seen the "reconnecting" event


  Scenario: Connections are reconnected when rtc is disconnected
    Given webrtc is intercepted by the testproxy

    Given "blue" is connected and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

    When "blue" creates a lobby
    And "blue" receives the network event "lobby" with the argument "19yrzmetd2bn7"

    When "yellow" connects to the lobby "19yrzmetd2bn7"
    And "blue" receives the network event "connected" with the argument "[Peer: prb67ouj837u]"

    When the connection between "yellow" and "blue" is interrupted until the first "disconnected" state

    And "blue" boardcasts "Goodbye, world!" over the reliable channel
    Then "yellow" receives the network event "reconnecting" with the argument "[Peer: 3t3cfgcqup9e]"
    And "yellow" receives the network event "reconnected" with the argument "[Peer: 3t3cfgcqup9e]"
    And "yellow" receives the network event "message" with the arguments "[Peer: 3t3cfgcqup9e]", "reliable" and "Goodbye, world!"
