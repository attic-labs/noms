<img src='doc/nommy_cropped_smaller.png' width='350' title='Nommy, the snacky otter'>

[Use Cases](#use-cases)&nbsp; | &nbsp;[Setup](#setup)&nbsp; | &nbsp;[Status](#status)&nbsp; | &nbsp;[Documentation](./doc/intro.md)&nbsp; | &nbsp;[Contact](#contact-us)
<br><br>

[![Build Status](https://travis-ci.org/attic-labs/noms.svg?branch=master)](https://travis-ci.org/attic-labs/noms)
[![Docker Build Status](https://img.shields.io/docker/build/noms/noms.svg)](https://hub.docker.com/r/noms/noms/)
[![GoDoc](https://godoc.org/github.com/attic-labs/noms?status.svg)](https://godoc.org/github.com/attic-labs/noms)

# Welcome

*Noms* is a decentralized database philosophically descendant from the Git version control system.

Like Git, Noms is:

* **Versioned:** By default, all previous versions of the database are retained. You can trivially track how the database evolved to its current state, easily and efficiently compare any two versions, or even rewind and branch from any previous version.
* **Synchronizable:** Instances of a single Noms database can be disconnected from each other for any amount of time, then later reconcile their changes efficiently and correctly.

Unlike Git, Noms is a database, so it also:

* Primarily **stores structured data**, not files and directories (see: [the Noms type system](https://github.com/attic-labs/noms/blob/master/doc/intro.md#types))
* **Scales well** to large amounts of data and concurrent clients
* Supports **atomic transactions** (a single instance of Noms is CP, but Noms is typically run in production backed by S3, in which case it is "[effectively CA](https://cloud.google.com/spanner/docs/whitepapers/SpannerAndCap.pdf)")
* Supports **efficient indexes** (see: [Noms prolly-trees](https://github.com/attic-labs/noms/blob/master/doc/intro.md#prolly-trees-probabilistic-b-trees))
* Features a **flexible query model** (see: [GraphQL](./go/ngql/README.md))

A Noms database can reside within a file system or in the cloud:

* The (built-in) [NBS](./go/nbs) `ChunkStore` implementation provides two back-ends which provide persistence for Noms databases: one for storage in a file system and one for storage in an S3 bucket.

Finally, because Noms is content-addressed, it yields a very pleasant programming model.

Working with Noms is ***declarative***. You don't `INSERT` new data, `UPDATE` existing data, or `DELETE` old data. You simply *declare* what the data ought to be right now. If you commit the same data twice, it will be deduplicated because of content-addressing. If you commit _almost_ the same data, only the part that is different will be written.

<br>

## Use Cases

#### [Decentralization](./doc/decent/about.md)

Because Noms is very good at sync, it makes a decent basis for rich, collaborative, fully-decentralized applications.

#### Mobile Offline-First Database

Embed Noms into mobile applications, making it easier to build offline-first, fully synchronizing mobile applications.

<br>

## Install

1. Download the latest release:
 - [**Linux**](https://github.com/attic-labs/noms/releases/download/latest/linux.zip)
 - [**Mac OS**](https://github.com/attic-labs/noms/releases/download/latest/osx.zip)
2. Unzip the directory somewhere and add it to your `$PATH`
3. Verify Noms is installed correctly:

```
$ noms version
format version: 7.18
built from <developer build>
```

<br>

## Run

Import some data:

```shell
go install github.com/attic-labs/noms/samples/go/csv/csv-import
curl 'https://data.cityofnewyork.us/api/views/kku6-nxdu/rows.csv?accessType=DOWNLOAD' > /tmp/data.csv
csv-import /tmp/data.csv /tmp/noms::nycdemo
```

Explore:

```shell
noms show /tmp/noms::nycdemo
```

Should show:

```go
struct Commit {
  meta: struct Meta {
    date: "2017-09-19T19:33:01Z",
    inputFile: "/tmp/data.csv",
  },
  parents: set {},
  value: [  // 236 items
    struct Row {
      countAmericanIndian: "0",
      countAsianNonHispanic: "3",
      countBlackNonHispanic: "21",
      countCitizenStatusTotal: "44",
      countCitizenStatusUnknown: "0",
      countEthnicityTotal: "44",
...
```

<br>

## Status

Nobody is working on this right now. You shouldn't rely on it unless you're willing to take over development yourself.

### Major Open Issues

These are the major things you'd probably want to fix before relying on this for most systems.

* Sync performance with long commit chains (https://github.com/attic-labs/noms/issues/2233)
* Migration (https://github.com/attic-labs/noms/issues/3363)
* Garbage Collection (https://github.com/attic-labs/noms/issues/3374)
* Query language
  * We started trying to hack in GraphQL but it's incomplete and maybe not the right thing. See: [ngql](./go/ngql/README.md)
* [Various other smaller bugs and improvements](https://github.com/attic-labs/noms/issues?q=is%3Aissue+is%3Aopen+label%3AP0)

<br>

## Learn More About Noms

For the decentralized web: [The Decentralized Database](doc/decent/about.md)

Learn the basics: [Technical Overview](doc/intro.md)

Tour the CLI: [Command-Line Interface Tour](doc/cli-tour.md)

Tour the Go API: [Go SDK Tour](doc/go-tour.md)

<br>

## Contact Us

Interested in using Noms? Awesome! We would be happy to work with you to help understand whether Noms is a fit for your problem. Reach out at:

- [Mailing List](https://groups.google.com/forum/#!forum/nomsdb)
- [Twitter](https://twitter.com/nomsdb)

## Licensing

Noms is open source software, licensed by Attic Labs, Inc. under the Apache License, Version 2.0.
