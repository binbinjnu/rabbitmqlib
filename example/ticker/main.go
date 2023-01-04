package main

import (
	"fmt"
	"go.slotsdev.info/server-group/rabbitmqlib/example/util"
	"time"
)

func main() {
	ticker1 := time.NewTicker(1 * time.Second)
	ticker2 := time.NewTicker(2 * time.Second)
	ticker3 := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case t1 := <-ticker1.C:
				fmt.Println("t1:", t1)
			case t2 := <-ticker2.C:
				fmt.Println("t2:", t2)
			case t3 := <-ticker3.C:
				fmt.Println("t3:", t3)
			}
		}
	}()
	util.WaitClose()
}
