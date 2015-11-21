@ls
Feature: ls command

  Scenario: I can list buckets
  	Given I have bucket "s3.barnybug.github.com"
    When I run "s3 ls"
    Then the output is "s3://s3.barnybug.github.com/\n"

  Scenario: I can list empty buckets
  	Given I have bucket "s3.barnybug.github.com"
    When I run "s3 ls s3://s3.barnybug.github.com/"
    Then the output is "\n0 files, 0 bytes\n"

  Scenario: I can list keys
  	Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" has key "apple" containing "123"
    And bucket "s3.barnybug.github.com" has key "banana" containing "123"
    When I run "s3 ls s3://s3.barnybug.github.com/apple"
    Then the output is "s3://s3.barnybug.github.com/apple\t3b\n\n1 files, 3 bytes\n"
