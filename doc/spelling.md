# Spelling in Noms

Many commands and APIs in Noms accept datastore, dataset, or value specifications as arguments. This document describes how to "spell" such specs.

## Spelling Datastores

Datastore names take the form:

```
<protocol>:<path>
```

The `path` part of the name is interpreted differently depending on the protocol:

- **http(s):** specs describe a remote datastore to be accessed over HTTP. In this case, the entire datastore spec is a normal http(s) URL. For example: `https://dev.noms.io/aa`.
- **ldb:** specs describe a local [LevelDB](https://github.com/google/leveldb)-backed datastore. In this case, the path component should be a relative or absolute path on disk to a directory in which to store the LevelDB data. For example: `ldb:~/noms-data`.
- **mem:** specs describe an ephemeral memory-backed datastore. In this case, the path component is not used and must be empty.

## Spelling Datasets

Dataset names take the form:

```
<datastore>:<dataset>
```

See [spelling datastores](#spelling-datastores) for how to build the `datastore` part of the name. The `dataset` part is just any string matching the regex `^[a-zA-Z0-9\-_/]+$`.

## Spelling Objects

Object names take the form:

```
<datastore>:<object-name>
```

See [spelling datastores](#spelling-datastores) for how to build the datastore part of the name.

The `object-name` part can be either a ref or a dataset name. If  `object-name` starts with `sha1-`, it will be interpreted as a ref. Otherwise it will be interpreted as a dataset name.

### Examples

```sh
# “foo” dataset at http://api.noms.io/-/aa
http://api.noms.io/-/aa:foo

# value sha1-123 at http://localhost:8000
http://localhost:8000/monkey:sha1-123

# “bonk” dataset at ldb:/foo/bar
ldb:/foo/bar:bonk
```