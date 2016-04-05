package main 

import (
	"flag"
	"os"
	"fmt"
	"strings"

	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/datasets"
	"github.com/attic-labs/noms/clients/noms/read_data"
)


func main() {
	//test number of args, format of object. Error message prints usage
	if (len(os.Args) != 2) {
		fmt.Println("Usage:\n")
		return
	}

	ds, r, err := ReadObject(os.Args[:1])

	if (!err) {
		fmt.Println("Usage:\n")
		return
	}

	//print representation... data set id for now
	fmt.Println(ds.ID())
}

