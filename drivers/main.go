package main

import (
	"context"
	"fmt"
	"github.com/eriksywu/go-async"
	"sync"
	"time"
)

func main() {
	//defer profile.Start(profile.MemProfile).Stop()
	pool := go_async.NewPool(10, context.Background())
	pool.Start()
	time.Sleep(5*time.Second)
	fmt.Println("======================================================")
	wg := sync.WaitGroup{}
	n := 10
	wg.Add(n)
	for i := 0; i < n; i++ {
		startWork(pool, &wg, i)
	}
	wg.Wait()
	cancelTask := pool.CancelAsync()
	cancelTask.Result()
	fmt.Println("wtf")
}

func startWork(pool *go_async.Pool, wg *sync.WaitGroup, agentNumber int) {
	ctx, _ := context.WithTimeout(context.Background(), 600* time.Second)
	go func() {
		defer wg.Done()
		i := 0
		queue := pool.GetQueue()

		for {
			select {
			case <- ctx.Done(): {
				return
			}
			case queue <- func() {
				fmt.Printf("executing job=%d for agent=%d\n", i, agentNumber)
				time.Sleep(1 * time.Second)
			}:
				i++
				fmt.Printf("agent=%d has finished sending jobs \n", agentNumber)
				if i == 10 {
					break
				}
			default:
				fmt.Printf("agent=%d is waiting to send job=%d \n", agentNumber, i)
				time.Sleep(1 * time.Second)
			}
		}

	}()

}
