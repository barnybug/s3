# s3

[![Build Status](https://secure.travis-ci.org/barnybug/s3.png)](http://travis-ci.org/barnybug/s3)

Swiss army pen-knife for Amazon S3.

- ls: List buckets or keys
- get: Download keys
- cat: Cat keys
- sync: Synchronise local to s3, s3 to local or s3 to s3

# Installation

    git clone https://github.com/barnybug/s3
    make all

# Setup

Set the environment variables:

    export AWS_ACCESS_KEY_ID=...
    export AWS_SECRET_ACCESS_KEY=...

# Usage

    s3 ls
    s3 get s3://bucket/path
    s3 cat s3://bucket/path | grep needle
    s3 sync localpath s3://bucket/path
    s3 sync s3://bucket/path
    s3 sync s3://bucket1/path s3://bucket2/otherpath
