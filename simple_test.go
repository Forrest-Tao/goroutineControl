package goroutineControl

import (
	"context"
	"github.com/panjf2000/ants/v2"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
	"log"
	"runtime"
	"sync"
	"testing"
	"time"
)

// runTaskDataGenerator 产生数据
func runTaskDataGenerator(dataChan chan int) {
	for i := 0; i < 100; i++ {
		dataChan <- i
	}

	close(dataChan)
}

// runNumGoroutineMonitor 协程数量监控
func runNumGoroutineMonitor() {
	log.Printf("协程数量->%d\n", runtime.NumGoroutine())

	for {
		select {
		case <-time.After(time.Second):
			log.Printf("协程数量->%d\n", runtime.NumGoroutine())
		}
	}
}

// runInfiniteTask 每来一个数据起协程处理任务
func runInfiniteTask(dataChan <-chan int) {
	var wg sync.WaitGroup

	for data := range dataChan {
		wg.Add(1)
		go func(data int) {
			defer wg.Done()

			// do something
			time.Sleep(3 * time.Second)
		}(data)
	}

	wg.Wait()
}

func TestRunInfiniteTask(t *testing.T) {
	dataChan := make(chan int)

	go runTaskDataGenerator(dataChan)
	go runNumGoroutineMonitor()

	runInfiniteTask(dataChan)
}

// runBoundedTask 起maxTaskNum个协程共同处理任务
func runBoundedTask(dataChan <-chan int, maxTaskNum int) {
	var wg sync.WaitGroup
	wg.Add(maxTaskNum)

	for i := 0; i < maxTaskNum; i++ {
		go func() {
			defer wg.Done()

			for data := range dataChan {
				func(data int) {

					// do something
					time.Sleep(3 * time.Second)
				}(data)
			}
		}()
	}

	wg.Wait()
}

func TestRunBoundedTask(t *testing.T) {
	dataChan := make(chan int)

	go runTaskDataGenerator(dataChan)
	go runNumGoroutineMonitor()

	runBoundedTask(dataChan, 10)
}

// runDynamicTask
// 最大同时运行maxTaskNum个任务处理数据
// 自定义令牌池维持maxTaskNum个令牌供竞争
func runDynamicTask(dataChan <-chan int, maxTaskNum int) {
	// 初始化令牌池
	tokenPool := make(chan struct{}, maxTaskNum)
	for i := 0; i < maxTaskNum; i++ {
		tokenPool <- struct{}{}
	}

	var wg sync.WaitGroup

	for data := range dataChan {
		// 先获取令牌，如果被消费完则阻塞等待其它任务返还令牌
		<-tokenPool

		wg.Add(1)
		go func(data int) {
			defer wg.Done()

			// 任务运行完成，返还令牌
			defer func() {
				tokenPool <- struct{}{}
			}()

			// do something
			time.Sleep(3 * time.Second)
		}(data)
	}

	wg.Wait()
}

func TestRunDynamicTask(t *testing.T) {
	dataChan := make(chan int)

	go runTaskDataGenerator(dataChan)
	go runNumGoroutineMonitor()

	runDynamicTask(dataChan, 50)
}

// runSemaphoreTask
// 最大同时运行maxTaskNum个任务处理数据
// 使用信号量维持maxTaskNum个信号
func runSemaphoreTask(dataChan <-chan int, maxTaskNum int64) {
	w := semaphore.NewWeighted(maxTaskNum)

	var wg sync.WaitGroup

	for data := range dataChan {
		// 先获取信号量，如果被消费完则阻塞等待信号量返还
		_ = w.Acquire(context.TODO(), 1)

		wg.Add(1)
		go func(data int) {
			defer wg.Done()

			// 运行完成返还信号量
			defer w.Release(1)

			// do something
			time.Sleep(3 * time.Second)
		}(data)
	}

	wg.Wait()
}

func TestRunSemaphoreTask(t *testing.T) {
	dataChan := make(chan int)

	go runTaskDataGenerator(dataChan)
	go runNumGoroutineMonitor()

	runSemaphoreTask(dataChan, 50)
}

// runRateLimitTask 限制每秒允许的最大协程数量，限流器的思路
func runRateLimitTask(dataChan <-chan int) {
	// 初始化令牌池
	tokenPool := make(chan struct{})
	go func() {
		for {
			select {
			// 动态控制令牌生成速度
			case <-time.After(time.Second):
				tokenPool <- struct{}{}
			}
		}
	}()

	var wg sync.WaitGroup

	for data := range dataChan {
		// 先获取令牌，如果被消费完则阻塞等待新令牌产生
		<-tokenPool

		wg.Add(1)
		go func(data int) {
			defer wg.Done()

			// do something
			time.Sleep(3 * time.Second)
		}(data)
	}

	wg.Wait()
}

func TestRunRateLimitTask(t *testing.T) {
	dataChan := make(chan int)

	go runTaskDataGenerator(dataChan)
	go runNumGoroutineMonitor()

	runRateLimitTask(dataChan)
}

// runRateLimitTask2 限制每秒允许的最大协程数量，使用官方限流器
func runRateLimitTask2(dataChan <-chan int) {
	// 初始化令牌池
	limit := rate.Every(time.Second) // 每秒一个
	limiter := rate.NewLimiter(limit, 10)

	var wg sync.WaitGroup

	for data := range dataChan {
		// 先获取令牌，如果被消费完则阻塞等待新令牌产生
		_ = limiter.Wait(context.TODO())

		wg.Add(1)
		go func(data int) {
			defer wg.Done()

			// do something
			time.Sleep(3 * time.Second)
		}(data)
	}

	wg.Wait()
}

func TestRunRateLimitTask2(t *testing.T) {
	dataChan := make(chan int)

	go runTaskDataGenerator(dataChan)
	go runNumGoroutineMonitor()

	runRateLimitTask2(dataChan)
}

// runGoroutinePoolTask 使用协程池动态管理协程数量
func runGoroutinePoolTask(dataChan <-chan int, maxTaskNum int) {
	p, _ := ants.NewPool(maxTaskNum)
	defer p.Release()

	var wg sync.WaitGroup

	for _ = range dataChan {
		wg.Add(1)

		// 提交任务，协程池动态管理数量，可以做更多的分配优化策略
		_ = p.Submit(func() {
			defer wg.Done()

			// do something
			time.Sleep(3 * time.Second)
		})

	}

	wg.Wait()
}

func TestRunGoroutinePoolTask(t *testing.T) {
	dataChan := make(chan int)

	go runTaskDataGenerator(dataChan)
	go runNumGoroutineMonitor()

	runGoroutinePoolTask(dataChan, 50)
}
