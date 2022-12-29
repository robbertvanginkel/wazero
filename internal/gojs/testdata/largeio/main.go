package largeio

import (
	"fmt"
	"io"
	"os"
)

func Main() {
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	fmt.Println(len(b))
}
