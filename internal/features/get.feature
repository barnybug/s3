@get
Feature: get command

  Scenario: I can get a file
  	Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "key" contains "123"
    When I run "s3 get s3://s3.barnybug.github.com/key"
    Then local file "key" has contents "123"

  Scenario: get from a non-existent bucket is an error
    When I run "s3 get s3://s3.barnybug.github.com/key"
    Then the exit code is 1

  Scenario: get a non-existent key is an error
  	Given I have bucket "s3.barnybug.github.com"
    When I run "s3 get s3://s3.barnybug.github.com/key"
    Then the exit code is 1

  Scenario: get local file is an error
  	Given I have bucket "s3.barnybug.github.com"
    When I run "s3 get ."
    Then the exit code is 1
