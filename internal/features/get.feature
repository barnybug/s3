@get
Feature: get command

  Scenario: I can get a file
  	Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "key" contains "123"
    When I run "s3 get s3://s3.barnybug.github.com/key"
    Then local file "key" has contents "123"
