package debug

import "fmt"

func Log(log string) {
	fmt.Printf("(@debug) %s\n", log)
}
