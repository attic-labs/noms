# Splore

Splore is a general-purpose debug UI for exploring noms data. It's deployed at http://splore.noms.io.

## Example

![splore and counter](screenshot.png)

All commands relative to `$GOPATH/src/github.com/attic-labs/noms`.

### Write some data
```
cd clients/counter
go build
./counter -ldb=/tmp/sploretest -ds=counter
./counter -ldb=/tmp/sploretest -ds=counter
```

### Start splore
```
cd clients/splore
npm start &
```

### Run noms-view
```
cd cmd/noms-view
go build
./noms-view start ../../clients/splore server=ldb:/tmp/sploretest
```

Then, navigate to the URL printed by noms-view, like `http://127.0.0.1:12345`.
