# Splore

Splore is a general-purpose debug UI for exploring noms data. It's deployed at http://splore.noms.io.

## Example

![splore and counter](screenshot.png)

All commands relative to the noms project:
```
cd $GOPATH/src/github.com/attic-labs/noms
```

### Write some data
```
cd clients/counter
go build
./counter -ldb=/tmp/sploretest -ds=counter
./counter -ldb=/tmp/sploretest -ds=counter
```

### Compile
```
cd ../splore
npm i  # only needs to be run once, or when package.json changes
```


#### One time (production) build
`npm run build`

#### Continuous (debug) build
`npm start &`

### Launch
```
cd ../../cmd/noms-view
go build
./noms-view serve ../../clients/splore store=ldb:/tmp/sploretest
```

Then, navigate to the URL printed by noms-view, like `http://127.0.0.1:12345`.
