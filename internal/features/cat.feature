@cat
Feature: cat command

  Scenario: I can cat a file
  	Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "aardvark" contains "AARDVARK"
    And bucket "s3.barnybug.github.com" key "apple" contains "APPLE"
    And bucket "s3.barnybug.github.com" key "banana" contains "BANANA"
    When I run "s3 cat s3://s3.barnybug.github.com/a"
    Then the output is "AARDVARKAPPLE"
