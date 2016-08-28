# Spelling in Noms

Many commands and APIs in Noms accept database, dataset, or value specifications as arguments. This document describes how to construct these specifications.

## Spelling Databases

database specifications take the form:

```
<protocol>[:<path>]
```

The `path` part of the name is interpreted differently depending on the protocol:

- **http(s)** specs describe a remote database to be accessed over HTTP. In this case, the entire database spec is a normal http(s) URL. For example: `https://dev.noms.io/aa`.
- **ldb** specs describe a local [LevelDB](https://github.com/google/leveldb)-backed database. In this case, the path component should be a relative or absolute path on disk to a directory in which to store the LevelDB data. For example: `ldb:/tmp/noms-data`.
  - In Go, `ldb:` can be ommitted (just `/tmp/noms-data` will work).
- **mem** specs describe an ephemeral memory-backed database. In this case, the path component is not used and must be empty.

## Spelling Datasets

Dataset specifications take the form:

```
<database>::<dataset>
```

See [spelling databases](#spelling-databases) for how to build the `database` part of the name. The `dataset` part is just any string matching the regex `^[a-zA-Z0-9\-_/]+$`.

Example datasets:

```
/tmp/test-db::my-dataset
ldb:/tmp/test-db::my-dataset
http://localhost:8000::registered-businesses
https://demo.noms.io/aa::music
```

## Spelling Values

Value specifications take the form:

```
<database>::<value-name><path>
```

See [spelling databases](#spelling-databases) for how to build the database part of the name.

The `value-name` part can be either a hash or a dataset name. If `value-name` matches the pattern `^#[0-9a-v]{32}$`, it will be interpreted as a hash otherwise it will be interpreted as a dataset name. See [spelling datasets](#spelling-datasets) for how to build the dataset part of the name.

The `path` part is relative to the hash or dataset provided in `value-name`. 

### Spelling Items in Structs
Elements of a Noms struct can be referenced using a period. For example, if the `value-name` is a dataset, then one can use `.value` to get the root of the data in the dataset. In this case `.value` selects the `value` field from the `Commit` struct at the top of the dataset. One could instead use `.meta` to select the `meta` struct from the `Commit` struct. The `value-name` does not need to be a dataset though, so if it is a hash that references a struct, the same notation still works: `#o38hugtf3l1e8rqtj89mijj1dq57eh4m.field`.

### Spelling Items in Lists, Maps, or Sets
Elements of a Noms list, map, or set can be retrieved using brackets. For example, if the dataset is a Noms map of number to struct then one could use `.value[42]` to get the Noms struct associated with the key 42. Similarly selecting the first element from a Noms list would be `.value[0]`. If the Noms map was keyed by string, then using `.value["10000002702001"]` would reference the Noms struct associated with key "10000002702001".

### Examples

```sh
# “sf-crime” dataset at https://demo.noms.io/cli-tour
https://demo.noms.io/cli-tour::sf-crime

# value o38hugtf3l1e8rqtj89mijj1dq57eh4m at https://localhost:8000
https://localhost:8000/monkey::#o38hugtf3l1e8rqtj89mijj1dq57eh4m

# “bonk” dataset at /foo/bar
/foo/bar::bonk

# from https://demo.noms.io/cli-tour, select the "sf-crime" dataset, the root
# value is a Noms map, select the value of the Noms map identified by string
# key "10000002702001", then from that resulting struct select the Address field
https://demo.noms.io/cli-tour::sf-crime.value["10000002702001"].Address
```

Be careful with shell escaping. Your shell might require escaping of the double quotes and other characters. e.g.:

```sh
> noms show https://demo.noms.io/cli-tour::sf-crime.value["10000002702001"].Address
Object not found: https://demo.noms.io/cli-tour::sf-crime.value[10000002702001].Address

> noms show https://demo.noms.io/cli-tour::sf-crime.value[\"10000002702001\"].Address
"3300 Block of 20TH ST"
```
