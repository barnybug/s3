@grep
Feature: grep command

  Scenario: I can grep a file
  	Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "banana" contains "BANANA"
    And bucket "s3.barnybug.github.com" key "carrot" contains "CARROT"
    And bucket "s3.barnybug.github.com" key "orange" contains "ORANGE"
    When I run "s3 grep O s3://s3.barnybug.github.com/"
    Then the output contains "s3://s3.barnybug.github.com/carrot:CARROT\n"
    Then the output contains "s3://s3.barnybug.github.com/orange:ORANGE\n"

  Scenario: grep requires at least 2 arguments
    When I run "s3 grep carrot"
    Then the exit code is 1

  Scenario: grep a non-existent bucket is an error
    When I run "s3 grep carrot s3://s3.barnybug.github.com/key"
    Then the exit code is 1

  Scenario: grep a non-existent key is an error
  	Given I have bucket "s3.barnybug.github.com"
    When I run "s3 grep carrot s3://s3.barnybug.github.com/key"
    Then the exit code is 1
