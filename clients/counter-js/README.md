# counter-js

counter-js uses noms to read and write a simple incrementing counter


## Getting Started

```
cd ../server
go build
./server -ldb=/tmp/counter-js &
cd ../splore
./build.py
npm run start
cd ../../cmd/noms-view
go build
./noms-view serve ../../clients/splore store="http://localhost:8000/" &
cd ../../clients/counter-js
npm install
npm run build
node dist/main.js http://localhost:8000/:counter-js
```

