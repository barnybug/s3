@get
Feature: get command

  Scenario: I can get a file
  	Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "path/key" contains "123"
    When I run "s3 get s3://s3.barnybug.github.com/path/key"
    Then local file "key" has contents "123"

  Scenario: I can get multiple files
    Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "aardvark" contains "AARDVARK"
    And bucket "s3.barnybug.github.com" key "apple" contains "APPLE"
    And bucket "s3.barnybug.github.com" key "banana" contains "BANANA"
    When I run "s3 get s3://s3.barnybug.github.com/a"
    Then local file "aardvark" has contents "AARDVARK"
    And local file "apple" has contents "APPLE"

  Scenario: I can get a directory
    Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "test/aardvark" contains "AARDVARK"
    And bucket "s3.barnybug.github.com" key "test/apple" contains "APPLE"
    When I run "s3 get s3://s3.barnybug.github.com/test"
    Then local file "test/aardvark" has contents "AARDVARK"
    And local file "test/apple" has contents "APPLE"

  Scenario: I can get a directory relatively
    Given I have bucket "s3.barnybug.github.com"
    And bucket "s3.barnybug.github.com" key "test/aardvark" contains "AARDVARK"
    When I run "s3 get s3://s3.barnybug.github.com/test/"
    Then local file "aardvark" has contents "AARDVARK"

  Scenario: get from a non-existent bucket is an error
    When I run "s3 get s3://s3.barnybug.github.com/key"
    Then the exit code is 1

  Scenario: get a non-existent key is an error
  	Given I have bucket "s3.barnybug.github.com"
    When I run "s3 get s3://s3.barnybug.github.com/key"
    Then the exit code is 1

  Scenario: get local file is an error
  	Given I have bucket "s3.barnybug.github.com"
    When I run "s3 get ."
    Then the exit code is 1
