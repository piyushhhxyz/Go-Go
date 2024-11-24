package main

import (
	"log"
	"sync"
	// "runtime"
)

type Counter struct {
	Mu  sync.Mutex
	Map map[string]int
}

func (c *Counter) Add(val int) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.Map["key"] = val
}

func main() {
	// Set the maximum number of CPUs to 1 to ensure the program uses only one core
	// runtime.GOMAXPROCS(1)

	c := Counter{Map: make(map[string]int)}
	wg := sync.WaitGroup{}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			c.Add(val)
		}(i)
	}

	wg.Wait()
	log.Println(c.Map["key"])
}