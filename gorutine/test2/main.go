package main

import "fmt"

func main() {
	resultCh := make(chan int)
	go func() {
		resultCh <- 42
	}()
	result := <-resultCh
	fmt.Println(result)
}
