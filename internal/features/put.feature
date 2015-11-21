@put
Feature: put command

  Scenario: I can put a file
  	Given I have bucket "s3.barnybug.github.com"
  	And local file "key" contains "abc"
    When I run "s3 put key s3://s3.barnybug.github.com/"
    Then bucket "s3.barnybug.github.com" has key "key" with contents "abc"

  Scenario: I can put multiple files
  	Given I have bucket "s3.barnybug.github.com"
  	And local file "apple" contains "APPLE"
  	And local file "banana" contains "BANANA"
    When I run "s3 put apple banana s3://s3.barnybug.github.com/"
    Then bucket "s3.barnybug.github.com" has key "apple" with contents "APPLE"
    Then bucket "s3.barnybug.github.com" has key "banana" with contents "BANANA"

  # TODO
  # Scenario: put a non-existent file is an error
  #   When I run "s3 put missing s3://s3.barnybug.github.com/"
  #   Then the exit code is 1

  Scenario: put to a non-existent bucket is an error
    Given local file "apple" contains "APPLE"
    When I run "s3 put apple s3://missing/"
    Then the exit code is 1
