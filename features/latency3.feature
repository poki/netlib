Feature: Country/region latency estimates

  Background:
    Given the "signaling" backend is running
    And the "testproxy" backend is running

  Scenario: latency is computed from requester to lobby peers
    Given these lobbies exist:
      | code          | game                                 | peers         | public |
      | 0qva9vyurwbbl | 123e4567-e89b-12d3-a456-426614174000 | {peerA,peerB} | true   |
    And these peers exist:
      | peer  | game                                 | country | region |
      | peerA | 123e4567-e89b-12d3-a456-426614174000 | US      | US-CA  |
      | peerB | 123e4567-e89b-12d3-a456-426614174000 | US      | US-CA  |
    And "blue" is connected as "1u8fw4aph5ypt" with country,region as "US","US-CA" and ready for game "123e4567-e89b-12d3-a456-426614174000"

    When "blue" requests lobbies with:
      """json
      {}
      """

    Then "blue" should have received only these lobbies:
      | code          | latency |
      | 0qva9vyurwbbl | 47      |


  Scenario: latency uses default when a peer has no country pair data
    Given these lobbies exist:
      | code          | game                                 | peers         | public |
      | 4z4an3whwhgvl | 223e4567-e89b-12d3-a456-426614174000 | {peerC,peerD} | true   |
    And these peers exist:
      | peer  | game                                 | country | region |
      | peerC | 223e4567-e89b-12d3-a456-426614174000 | US      | US-CA  |
      | peerD | 223e4567-e89b-12d3-a456-426614174000 | ZZ      | null   |
    And "green" is connected as "1u8fw4aph5ypt" with country,region as "US","US-CA" and ready for game "223e4567-e89b-12d3-a456-426614174000"

    When "green" requests lobbies with:
      """json
      {}
      """

    Then "green" should have received only these lobbies:
      | code          | latency |
      | 4z4an3whwhgvl | 148     |


  Scenario: latency uses default when requester has no country
    Given these lobbies exist:
      | code          | game                                 | peers   | public |
      | 3l6t2w7xk6w0y | 323e4567-e89b-12d3-a456-426614174000 | {peerE} | true   |
    And these peers exist:
      | peer  | game                                 | country | region |
      | peerE | 323e4567-e89b-12d3-a456-426614174000 | US      | US-CA  |
    And "orange" is connected as "1u8fw4aph5ypt" with country,region as "XX","XX" and ready for game "323e4567-e89b-12d3-a456-426614174000"

    When "orange" requests lobbies with:
      """json
      {}
      """

    Then "orange" should have received only these lobbies:
      | code          | latency |
      | 3l6t2w7xk6w0y | 250     |
