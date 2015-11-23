@sync
Feature: sync command

  Scenario: I can sync local to S3
  	Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "apple" contains "orange"
  	And local file "apple" contains "APPLE"
  	And local file "banana" contains "BANANA"
    When I run "s3 sync . s3://s3.barnybug.github.com/"
    Then bucket "s3.barnybug.github.com" has key "apple" with contents "APPLE"
    Then bucket "s3.barnybug.github.com" has key "banana" with contents "BANANA"
    And the output contains "U apple\n"
    And the output contains "A banana\n"
    And the output contains "1 added 0 deleted 1 updated 0 unchanged\n"

  Scenario: I can sync local to S3 deletes
    Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "banana" contains "BANANA"
    And local file "apple" contains "APPLE"
    When I run "s3 sync --delete . s3://s3.barnybug.github.com/"
    Then bucket "s3.barnybug.github.com" has key "apple" with contents "APPLE"
    And the output contains "A apple\n"
    And the output contains "D banana\n"
    And the output contains "1 added 1 deleted 0 updated 0 unchanged\n"

  Scenario: I can sync S3 to local
    Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "apple" contains "APPLE"
    And bucket "s3.barnybug.github.com" key "banana" contains "BANANA"
    When I run "s3 sync s3://s3.barnybug.github.com/ folder1"
    Then local file "folder1/apple" has contents "APPLE"
    Then local file "folder1/banana" has contents "BANANA"
    And the output contains "A apple\n"
    And the output contains "A banana\n"
    And the output contains "2 added 0 deleted 0 updated 0 unchanged\n"

  Scenario: I can sync S3 to S3
    Given I have bucket "s3.barnybug.github.com"
    And I have bucket "s3b.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "apple" contains "APPLE"
    And bucket "s3.barnybug.github.com" key "banana" contains "BANANA"
    And bucket "s3b.barnybug.github.com" key "banana" contains "BOO"
    When I run "s3 sync s3://s3.barnybug.github.com/ s3://s3b.barnybug.github.com/"
    Then bucket "s3b.barnybug.github.com" has key "apple" with contents "APPLE"
    And bucket "s3b.barnybug.github.com" has key "banana" with contents "BANANA"
    And the output contains "A apple\n"
    And the output contains "U banana\n"
    And the output contains "1 added 0 deleted 1 updated 0 unchanged\n"

  Scenario: sync needs 2 parameters
    When I run "s3 sync s3://s3.barnybug.github.com/"
    Then the exit code is 1

  Scenario: sync needs only 2 parameters
    When I run "s3 sync s3://s3.barnybug.github.com/ s3://s3b.barnybug.github.com/ s3://s3c.barnybug.github.com/"
    Then the exit code is 1

  Scenario: sync from a non-existent bucket is an error
    Given I have bucket "s3b.barnybug.github.com"
    When I run "s3 sync s3://s3.barnybug.github.com/ s3://s3b.barnybug.github.com/"
    Then the exit code is 1

  Scenario: sync to a non-existent bucket is an error
    Given I have bucket "s3.barnybug.github.com"
    When I run "s3 sync s3://s3.barnybug.github.com/ s3://s3b.barnybug.github.com/"
    Then the exit code is 1
