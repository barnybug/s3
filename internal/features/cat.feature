@cat
Feature: cat command

  Scenario: I can cat files
  	Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "aardvark" contains "AARDVARK"
    And bucket "s3.barnybug.github.com" key "apple" contains "APPLE"
    And bucket "s3.barnybug.github.com" key "banana" contains "BANANA"
    When I run "s3 cat s3://s3.barnybug.github.com/a"
    Then the output contains "AARDVARK"
    And the output contains "APPLE"

  Scenario: cat a non-existent bucket is an error
    When I run "s3 cat s3://s3.barnybug.github.com/key"
    Then the exit code is 1

  Scenario: cat a non-existent key is an error
  	Given I have bucket "s3.barnybug.github.com"
    When I run "s3 cat s3://s3.barnybug.github.com/key"
    Then the exit code is 1
