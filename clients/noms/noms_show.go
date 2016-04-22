package noms 

import (
	"os"
	"fmt"

	/*"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/dataset"*/
)


func main() {
	//test number of args, format of object. Error message prints usage
	if (len(os.Args) != 2) {
		fmt.Println("Usage:\n")
		return
	}

	ds, r, err, isDs := ReadObject(os.Args[1])

	if (!err) {
		fmt.Println("Usage:\n")
		return
	}

	//print representation... data set id for now
	if (isDs) {
		fmt.Println(ds.ID())
	} else {
		fmt.Println(r)
	}
}

