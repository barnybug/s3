@sync
Feature: sync command

  Scenario: I can sync a file
  	Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "apple" contains "orange"
  	And local file "apple" contains "APPLE"
  	And local file "banana" contains "BANANA"
    When I run "s3 sync . s3://s3.barnybug.github.com/"
    Then bucket "s3.barnybug.github.com" has key "apple" with contents "APPLE"
    Then bucket "s3.barnybug.github.com" has key "banana" with contents "BANANA"
    Then the output contains "U apple\n"
    Then the output contains "A banana\n"
    Then the output contains "1 added 0 deleted 1 updated 0 unchanged\n"
