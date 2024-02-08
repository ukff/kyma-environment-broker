package internal

import (
	"fmt"
	"sync"
)

var mu sync.Mutex

func Log(log string) {
	mu.Lock()
	defer mu.Unlock()
	fmt.Printf("(@debug) %s\n", log)
}
