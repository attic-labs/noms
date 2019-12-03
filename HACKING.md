# Prerequisites

* [Go 1.13 or later](https://golang.org/dl/)
* Mac or Linux (Noms isn't currently supported on Windows)

# Get

```
git clone https://github.com/attic-labs/noms
```

# Build

```
cd noms
go build ./cmd/noms
```

# Test

```
cd noms
go test ./go/...
go test ./cmd/...
```

# Release

Travis automatically creates releases for tagged versions, so the following should do it:

```
git tag latest -f
git push origin latest
```
