@mb
Feature: mb command

  Scenario: I can create a bucket
    When I run "s3 mb s3.barnybug.github.com"
    Then the bucket "s3.barnybug.github.com" exists

  Scenario: creating a duplicate bucket is an error
  	Given I have bucket "s3.barnybug.github.com"
    When I run "s3 mb s3.barnybug.github.com"
    Then the exit code is 1
