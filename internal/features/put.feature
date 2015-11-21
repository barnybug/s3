@put
Feature: put command

  Scenario: I can put a file
  	Given I have bucket "s3.barnybug.github.com"
  	And local file "key" contains "abc"
    When I run "s3 put key s3://s3.barnybug.github.com/"
    Then bucket "s3.barnybug.github.com" has key "key" with contents "abc"
