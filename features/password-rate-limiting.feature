Feature: Rate limiting for password attempts

  Background:
    Given the "signaling" backend is running

  Scenario: Rate limiting blocks too many password attempts from same IP
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "blue" creates a lobby with these settings:
      """json
      {
        "password": "foobar"
      }
      """
    And "blue" receives the network event "lobby" with the argument "19yrzmetd2bn7"

    # Make several failed password attempts
    When "yellow" tries to connect to the lobby "19yrzmetd2bn7" with the password "wrong1"
    Then "yellow" failed to join the lobby
    And the latest error for "yellow" is "invalid password"

    When "yellow" tries to connect to the lobby "19yrzmetd2bn7" with the password "wrong2"
    Then "yellow" failed to join the lobby
    And the latest error for "yellow" is "invalid password"

    When "yellow" tries to connect to the lobby "19yrzmetd2bn7" with the password "wrong3"
    Then "yellow" failed to join the lobby
    And the latest error for "yellow" is "invalid password"

    When "yellow" tries to connect to the lobby "19yrzmetd2bn7" with the password "wrong4"
    Then "yellow" failed to join the lobby
    And the latest error for "yellow" is "invalid password"

    When "yellow" tries to connect to the lobby "19yrzmetd2bn7" with the password "wrong5"
    Then "yellow" failed to join the lobby
    And the latest error for "yellow" is "invalid password"

    # The 6th attempt should be rate limited
    When "yellow" tries to connect to the lobby "19yrzmetd2bn7" with the password "wrong6"
    Then "yellow" failed to join the lobby
    And the latest error for "yellow" is "too many password attempts"