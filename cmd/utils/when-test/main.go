package main

import (
	"fmt"
	"github.com/cj123/assetto-server-manager/pkg/when"
	"time"
)

func main() {
	ch := make(chan struct{})

	stopCh, err := when.When(time.Date(2019, 12, 9, 16, 38, 42, 03330, time.Local), func() {
		fmt.Println("HI it's ", time.Now())
	})

	if err == nil {
		go func() {
			time.Sleep(time.Second * 10)
			stopCh <- struct{}{}
		}()
	}

	when.When(time.Date(2019, 12, 9, 16, 38, 02, 4453, time.Local), func() {
		fmt.Println("HI 2 it's ", time.Now())
	})

	<-ch
}
