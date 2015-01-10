# s3

[![Build Status](https://secure.travis-ci.org/barnybug/s3.png)](http://travis-ci.org/barnybug/s3)
[![Gobuild Download](http://gobuild.io/badge/github.com/barnybug/s3/download.png)](http://gobuild.io/github.com/barnybug/s3)

Swiss army pen-knife for Amazon S3.

- ls: List buckets or keys
- get: Download keys
- cat: Cat keys
- sync: Synchronise local to s3, s3 to local or s3 to s3
- rm: Delete keys
- mb: Create buckets
- rb: Delete buckets

# Installation

Installation is super-easy (there's no need to install anything), just go to
[gobuild.io](http://gobuild.io/download/github.com/barnybug/s3) and the s3 binary will be built on demand for your platform (Linux, Mac or Windows):
http://gobuild.io/download/github.com/barnybug/s3

Unzip the .zip file you've downloaded, and you can run s3 from there:

    ./s3 -help

However, if you do want to build from source, you'll need go 1.2 installed, then:

    go get github.com/barnybug/s3

# Setup

Set the environment variables:

    export AWS_ACCESS_KEY_ID=...
    export AWS_SECRET_ACCESS_KEY=...

# Usage

List buckets:

    s3 ls

List keys in a bucket under a prefix:

    s3 ls s3://bucket/prefix

Download all the contents (recursively) under the path to local:

    s3 get s3://bucket/path

Cat (stream to stdout) all the contents under the path:

    s3 cat s3://bucket/path | grep needle

Synchronise localpath to an s3 bucket:

    s3 sync localpath s3://bucket/path

Synchronise an s3 bucket to localpath:

    s3 sync s3://bucket/path localpath

Synchronise an s3 bucket to another s3 bucket:

    s3 sync s3://bucket1/path s3://bucket2/otherpath

Recursively remove all keys under a path:

    s3 rm s3://bucket/path

Create a bucket:

    s3 mb bucket

Delete a bucket:

    s3 rb bucket
