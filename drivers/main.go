package main

import (
	"context"
	"fmt"
	"github.com/eriksywu/go-async"
	"sync"
	"time"
)

func main() {
	pool := go_async.NewPool(10, context.Background())
	pool.Start()
	wg := sync.WaitGroup{}
	wg.Add(10)
	for i := 0; i < 10; i++ {
		startWork(pool, wg, i)
	}
	wg.Wait()
	cancelTask := pool.CancelAsync()
	cancelTask.Result()
	fmt.Println("wtf")
}

func startWork(pool *go_async.Pool, wg sync.WaitGroup, agentNumber int) {
	go func() {
		defer wg.Done()
		i := 0
		select {
		case pool.GetQueue() <- func() {
			fmt.Printf("agent=%d sent job=%d \n", agentNumber, i)
			time.Sleep(1 * time.Second)
		}:
			i++
			fmt.Printf("agent=%d has finished sending jobs \n", agentNumber)
			if i == 100 {
				break
			}
		default:
			fmt.Println("do nothing")
			time.Sleep(1 * time.Second)
		}
	}()

}
