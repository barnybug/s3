@rm
Feature: rm command

  Scenario: I can remove a key
  	Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "key" contains "1"
    When I run "s3 rm s3://s3.barnybug.github.com/key"
    Then bucket "s3.barnybug.github.com" key "key" does not exist
