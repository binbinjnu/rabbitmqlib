package util

import (
	"fmt"
	"os"
	"os/signal"
)

func WaitClose() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, os.Kill)
	s := <-c
	fmt.Println("Got signal:", s)
}
