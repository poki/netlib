Feature: Geo-based latency2 estimates

  Background:
    Given the "signaling" backend is running
    And the "testproxy" backend is running

  Scenario: latency2 is computed using peer geo persisted on connect
    Given "red" is connected as "1u8fw4aph5ypt" with lat,lon as 10,20 and ready for game "323e4567-e89b-12d3-a456-426614174000"
    And "yellow" is connected as "h5yzwyizlwao" with lat,lon as 30,40 and ready for game "323e4567-e89b-12d3-a456-426614174000"
    And "red,yellow" are joined in a public lobby
    And "blue" is connected as "3t3cfgcqup9e" with lat,lon as 50,60 and ready for game "323e4567-e89b-12d3-a456-426614174000"

    When "blue" requests lobbies with:
      """json
      {}
      """

    Then "blue" should have received only these lobbies:
      | code          | latency2 |
      | 19yrzmetd2bn7 | 69       |

  Scenario: latency2 is computed from requester to lobby peers
    Given these lobbies exist:
      | code          | game                                 | peers         | public |
      | 0qva9vyurwbbl | 123e4567-e89b-12d3-a456-426614174000 | {peerA,peerB} | true   |
    And these peers exist:
      | peer  | game                                 | geo    |
      | peerA | 123e4567-e89b-12d3-a456-426614174000 | 10, 20 |
      | peerB | 123e4567-e89b-12d3-a456-426614174000 | 30, 40 |
    And "blue" is connected as "1u8fw4aph5ypt" with lat,lon as 50,60 and ready for game "123e4567-e89b-12d3-a456-426614174000"

    When "blue" requests lobbies with:
      """json
      {}
      """

    Then "blue" should have received only these lobbies:
      | code          | latency2 |
      | 0qva9vyurwbbl | 69       |


  Scenario: latency2 is undefined when requester has no geo
    Given these lobbies exist:
      | code          | game                                 | peers   | public |
      | 0qva9vyurwbbl | 223e4567-e89b-12d3-a456-426614174000 | {peerC} | true   |
    And these peers exist:
      | peer  | game                                 | geo    |
      | peerC | 223e4567-e89b-12d3-a456-426614174000 | 10, 10 |
    And "green" is connected as "1u8fw4aph5ypt" and ready for game "223e4567-e89b-12d3-a456-426614174000"

    When "green" requests lobbies with:
      """json
      {}
      """

    Then "green" should have received only these lobbies:
      | code          | latency2  |
      | 0qva9vyurwbbl | undefined |
