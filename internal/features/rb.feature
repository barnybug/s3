@rb
Feature: rb command

  Scenario: I can remove a bucket
  	Given I have bucket "s3.barnybug.github.com"
    When I run "s3 rb s3.barnybug.github.com"
    Then the bucket "s3.barnybug.github.com" does not exist
