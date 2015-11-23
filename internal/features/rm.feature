@rm
Feature: rm command

  Scenario: I can remove a key
  	Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "key" contains "1"
    When I run "s3 rm s3://s3.barnybug.github.com/key"
    Then bucket "s3.barnybug.github.com" key "key" does not exist

  Scenario: I can remove multiple keys
  	Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "apple" contains "1"
    And bucket "s3.barnybug.github.com" key "avocado" contains "1"
    And bucket "s3.barnybug.github.com" key "banana" contains "1"
    When I run "s3 rm s3://s3.barnybug.github.com/a"
    Then bucket "s3.barnybug.github.com" key "apple" does not exist
    And bucket "s3.barnybug.github.com" key "avocado" does not exist
    And bucket "s3.barnybug.github.com" key "banana" exists

  Scenario: rm from a non-existent bucket is an error
    When I run "s3 rm s3://s3.barnybug.github.com/key"
    Then the exit code is 1

  Scenario: rm a non-existent key is an error
  	Given I have bucket "s3.barnybug.github.com"
    When I run "s3 rm s3://s3.barnybug.github.com/key"
    Then the exit code is 1

  Scenario: rm a local file is an error
  	Given local file "localfile" contains "abc"
    When I run "s3 rm localfile"
    Then the exit code is 1
    And local file "localfile" has contents "abc"
