@rb
Feature: rb command

  Scenario: I can remove a bucket
  	Given I have bucket "s3.barnybug.github.com"
    When I run "s3 rb s3.barnybug.github.com"
    Then the bucket "s3.barnybug.github.com" does not exist

  Scenario: removing a non-existent bucket is an error
    When I run "s3 rb s3.barnybug.github.com"
    Then the exit code is 1

  Scenario: removing a bucket containing keys is an error
    Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "path/key" contains "123"
    When I run "s3 rb s3.barnybug.github.com"
    Then the exit code is 1
