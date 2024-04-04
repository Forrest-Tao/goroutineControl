package goroutineControl

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	waiting := make(chan struct{})
	go func() {
		<-sig
		fmt.Println("ready to clos0e")
		time.Sleep(3 * time.Second)
		waiting <- struct{}{}
	}()

	go func() {
		for {
			select {
			default:
				time.Sleep(time.Millisecond * 100)
				fmt.Println("working...")
			}
		}
	}()
	<-waiting
}
