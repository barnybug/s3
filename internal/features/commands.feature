Feature: get
  Scenario: I can get a file
    Given I have empty bucket "s3.barnybug.github.com"
    And the bucket "s3.barnybug.github.com" has a key "key" with contents "123"
    When I run "s3 get s3://s3.barnybug.github.com/key"
    Then local file "key" has contents "123"
