Feature: List
  Scenario: Define list variables
    Given a file named "main.ein" with:
    """
    l : [Number]
    l = [42]

    main : Number -> [Number]
    main x = [42]
    """
    When I successfully run `ein build main.ein`
    And I successfully run `sh -c ./a.out`
    Then the stdout from "sh -c ./a.out" should contain exactly "42"

  Scenario Outline: Use list case expressions with single alternatives
    Given a file named "main.ein" with:
    """
    main : Number -> [Number]
    main x = [<case expression>]
    """
    When I successfully run `ein build main.ein`
    And I successfully run `sh -c ./a.out`
    Then the stdout from "sh -c ./a.out" should contain exactly "42"
    Examples:
      | case expression                     |
      | case [42] of [42] -> 42             |
      | case [42] of [y] -> 42              |
      | case [42] of [y] -> y               |
      | case [42, 42] of [42, 42] -> 42     |
      | case [42, 42] of [y, 42] -> y       |
      | case [42, 42] of [42, y] -> y       |
      | case [42] of [x, ...xs] -> 42       |
      | case [42] of [x, ...xs] -> x        |
      | case [42, 42] of [x, y, ...xs] -> y |

  Scenario: Use list case expressions with multiple alternatives
    Given a file named "main.ein" with:
    """
    main : Number -> [Number]
    main x =
      case [42] of
        [42, 0] -> [13]
        [y] -> [42]
    """
    When I successfully run `ein build main.ein`
    And I successfully run `sh -c ./a.out`
    Then the stdout from "sh -c ./a.out" should contain exactly "42"

  Scenario: Use complex list case expressions
    Given a file named "main.ein" with:
    """
    main : Number -> [Number]
    main x =
      case [1, 2, 3] of
        [x, 2, 4] -> [13]
        [1, x, 4] -> [13]
        [1, 2, x] -> [42]
    """
    When I successfully run `ein build main.ein`
    And I successfully run `sh -c ./a.out`
    Then the stdout from "sh -c ./a.out" should contain exactly "42"
