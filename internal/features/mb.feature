@mb
Feature: mb command

  Scenario: I can create a bucket
    When I run "s3 mb s3.barnybug.github.com"
    Then the bucket "s3.barnybug.github.com" exists
