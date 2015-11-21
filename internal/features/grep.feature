@grep
Feature: grep command

  Scenario: I can grep a file
  	Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "apple" contains "APPLE"
    And bucket "s3.barnybug.github.com" key "banana" contains "BANANA"
    And bucket "s3.barnybug.github.com" key "carrot" contains "CARROT"
    When I run "s3 grep BANANA s3://s3.barnybug.github.com/"
    Then the output is "s3://s3.barnybug.github.com/banana\n"
