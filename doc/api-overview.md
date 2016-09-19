# Noms API Overview

Noms provides SDKs for Go and JavaScript development. This overview introduces the basic types and APIs available to Go and JavaScript developers.

## Generated API Documentation

The API documentation that is derived from the source code automatically can be found by following these links: 

* [Go Docs](https://godoc.org/github.com/attic-labs/noms)
* [JavaScript Docs](https://docs.noms.io/js)

## Noms Setup

If you haven't already done so, please read the [Intro to Noms](intro.md). You should also review the [Go Tour](go-tour.md) or the [JavaScript Tour](js-tour.md) for help getting the appropriate SDK installed.

## Specifying Data

In order to store or retrieve data you will need to connect to a database and specify a dataset. Noms has `paths` for specifying resources like databases, datasets and values. The [Spelling in Noms](spelling.md) documentation explains the finer details on how to construct these `paths`. The Noms [go/spec package](https://godoc.org/github.com/attic-labs/noms/go/spec) and [JavaScript specs.js](https://github.com/attic-labs/noms/blob/master/js/noms/src/specs.js) are the source code locations to review for parsing and building `paths`. Several of the [samples](https://github.com/attic-labs/noms/tree/master/samples) use `paths` specified as command-line arguments and can be used as examples for properly parsing, building and checking for the errors that are returned from these APIs.

## Data Versioning

Once you have a database/dataset specified, you can store and retrieve data. A dataset, like a `git` repository has a `HEAD` revision - or a marker for the current version of the data. You can access a Noms dataset `HEAD` using the dataset API - [Head](https://godoc.org/github.com/attic-labs/noms/go/dataset#Dataset.Head). If you want to set a different version of your data to be `HEAD` then you would use the dataset [`Commit`](https://godoc.org/github.com/attic-labs/noms/go/dataset#Dataset.Commit) method.

## Noms Types
Noms supports [several data types](intro.md#types). Most of these types will be familiar and behave as you expect. Noms types are [immutable](https://en.wikipedia.org/wiki/Immutable_object). For example, at the end of the last section when calling `Commit` on dataset, the dataset you were holding was not updated. Instead `Commit` returned a new `dataset` for the new `HEAD` you created.

### Noms Values

All Noms values implement the [Value](https://godoc.org/github.com/attic-labs/noms/go/types#Value) interface. Noms lists, sets and maps are all collections of Noms values - but they are also Noms values themselves and can therefore be nested to create complex data structures. Noms includes value implementations for booleans, strings, structs and numbers too.

### Noms Lists

Lists are one of the value collection types supported in Noms - [Go](https://godoc.org/github.com/attic-labs/noms/go/types#List) and [JavaScript](https://docs.noms.io/js/#list). Lists are similar to arrays or vectors in most languages. 

- `Empty` returns a boolean and enables you to test if a list is empty
- `Len` returns the number of values in the list 
- `Get` and `Set` enables retrieving and storing, respectively, Noms values at specific indices.
- `Remove` and `RemoveAt` allow for deleting values from a Noms list by index.
- `Append` allows for adding additional Noms values to the end of the list.
- `Insert` allows for adding Noms values at a specific index.
- Similarly `Splice` supports removing items at an index and inserting Noms values at the same index.
- Iterating over Noms values in the list can be done by calling `Iterator` or `IteratorAt` and using the [ListIterator](https://godoc.org/github.com/attic-labs/noms/go/types#ListIterator) returned to call `Next` to retrieve the values in order by their index. These methods for iterating are recommended over a typical C-style `for` loop calling `Get` on an increasing index value.
- Alternatively you can execute code on each item by using `Iter` or `IterAll`.
- `Diff` provides an implementation to easily calculate the differences between two Noms lists

### Noms Sets

If you need to store and retrieve Noms values by their value instead of by their index then you could use a Noms set - [Go](https://godoc.org/github.com/attic-labs/noms/go/types#Set) and [JavaScript](https://docs.noms.io/js/#set). Since the storage and retrieval is based on the contents of the Noms values this eliminates duplicate values and the potential of having to scan Noms lists to determine if a value already exists in the collection.

### Noms Maps

Noms provides a data structure that is similar to a hash table - called `Map` - [Go](https://godoc.org/github.com/attic-labs/noms/go/types#Map) and [JavaScript](https://docs.noms.io/js/#map). Noms maps store keys and values. The keys and values in Noms maps are Noms values. Noms maps provide many similar methods to Noms lists and sets:

- `Empty` can be used to test for an empty Noms map
- `Len` returns the number of keys in the Noms map
- `Has` can be used to test if a provided key is present in the Noms map
- `Get` returns the Noms value for the provided key 
- `MaybeGet` is like `Get` but also returns a boolean to indicate if a value exists for the provided key
- `Set` stores the provided value for the given key
- `Remove` deletes the provided key and value from the Noms map
- `Diff` provides an implementation to easily calculate the differences between two Noms maps
- `Iter`, `IterAll`, `IterFrom` all execute the provided function on the entries in the Noms map
- `First` returns the first key/value in the Noms map
- `Last` returns the last key/value pair in the Noms map

### Noms Structs

Noms structs provide a data structure to store and retrieve Noms values similar to Noms maps but the keys are strings and the values are Noms values. Noms structs provide type checking to ensure the keys being used and the values being set are as expected. 

- `Get`
- `MaybeGet`
- `Set`
- `Diff`

*TODO: write about Noms structs here*

## JavaScript Specifics

### Flow, Promises, Unicorns, Oh My!

Noms uses several JavaScript packages to improve source code quality, readability and performance.  You can see the dependencies by reviewing [package.json](https://github.com/attic-labs/noms/blob/master/js/noms/package.json). Many of these will not affect you, but a few definitely will. The usage of [Promises](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise) and [Flow](https://flowtype.org/) type annotations will be noticeable in every JavaScript file and most APIs. If you are not already familiar with these projects or concepts it is worth your time to go learn the basics as it will make it easier for you to develop using the Noms JavaScript SDK. You do not need to use Flow in your projects if you are building a JavaScript client that depends on Noms though.

## Go Specifics

### Go `Maybe`?

Some of the Noms Go APIs include functions that are prefixed with `Maybe`. These are usually functions that allow for the caller to check for error or does-exist conditions. For example, [`dataset`](https://godoc.org/github.com/attic-labs/noms/go/dataset) includes [`Head`](https://godoc.org/github.com/attic-labs/noms/go/dataset#Dataset.Head) and [`MaybeHead`](https://godoc.org/github.com/attic-labs/noms/go/dataset#Dataset.MaybeHead). If you are not sure if your dataset has a `Head` you can call `MaybeHead` and check the second return value (a `bool`) to know if a `Head` exists. Another place this pattern can be seen is in the [Noms map API](https://godoc.org/github.com/attic-labs/noms/go/types#Map) with [`MaybeGet`](https://godoc.org/github.com/attic-labs/noms/go/types#Map.MaybeGet) and similarly in a Noms struct with `MaybeGet`.

### Go Marshalling

*TODO: include info on go/marshall package and usage - or maybe it should be added to the Go SDK Tour?*
