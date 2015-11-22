@put
Feature: put command

  Scenario: I can put a file
  	Given I have bucket "s3.barnybug.github.com"
  	And local file "path/key" contains "abc"
    When I run "s3 put path/key s3://s3.barnybug.github.com/"
    Then bucket "s3.barnybug.github.com" has key "key" with contents "abc"

  Scenario: I can put rename file
    Given I have bucket "s3.barnybug.github.com"
    And local file "path/key" contains "abc"
    When I run "s3 put path/key s3://s3.barnybug.github.com/key2"
    Then bucket "s3.barnybug.github.com" has key "key2" with contents "abc"

  Scenario: I can put multiple files
  	Given I have bucket "s3.barnybug.github.com"
  	And local file "apple" contains "APPLE"
  	And local file "banana" contains "BANANA"
    When I run "s3 put apple banana s3://s3.barnybug.github.com/path/"
    Then bucket "s3.barnybug.github.com" has key "path/apple" with contents "APPLE"
    Then bucket "s3.barnybug.github.com" has key "path/banana" with contents "BANANA"

  Scenario: I can put a directory
    Given I have bucket "s3.barnybug.github.com"
    And local file "top/path/key" contains "abc"
    When I run "s3 put top/path s3://s3.barnybug.github.com/"
    Then bucket "s3.barnybug.github.com" has key "path/key" with contents "abc"

  Scenario: I can put a directory's contents
    Given I have bucket "s3.barnybug.github.com"
    And local file "top/path/key" contains "abc"
    When I run "s3 put top/path/ s3://s3.barnybug.github.com/here/"
    Then bucket "s3.barnybug.github.com" has key "here/key" with contents "abc"

  Scenario: put a non-existent file is an error
    When I run "s3 put missing s3://s3.barnybug.github.com/"
    Then the exit code is 1

  Scenario: put to a non-existent bucket is an error
    Given local file "apple" contains "APPLE"
    When I run "s3 put apple s3://missing/"
    Then the exit code is 1

  Scenario: put to a local file is an error
    Given local file "key" contains "abc"
    When I run "s3 put key path"
    Then the exit code is 1
