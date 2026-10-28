package main

import (
	"fmt"
	"sync"
)

func MyPrintSlice(name string, slice []int) {
	fmt.Print(name)
	for i, number := range slice {
		if len(slice)/2 == i {
			fmt.Print("| ")
		}
		fmt.Print(number, " ")
	}
	fmt.Println()
}

func MyPrintResultCh(ch <-chan int) {

	for i := 0; true; i++ {
		value, ok := <-ch
		if !ok {
			return
		}
		fmt.Print("Сумма половины среза ", i+1, ": ")
		fmt.Println(value)
	}
}

func Sum(slice []int) int {
	result := 0
	for _, number := range slice {
		result += number
	}
	return result
}

func AmountHalfSlice(wg *sync.WaitGroup, ch chan<- int, slice []int) {
	defer wg.Done()
	ch <- Sum(slice)
}

func main() {
	var wg sync.WaitGroup
	var slice []int = []int{2, 6, 7, 3, 6, 8}
	MyPrintSlice("Числа среза: ", slice)
	ch := make(chan int)
	wg.Add(2)
	go AmountHalfSlice(&wg, ch, slice[:len(slice)/2])
	go AmountHalfSlice(&wg, ch, slice[len(slice)/2:])
	go func() {
		wg.Wait()
		close(ch)
	}()
	MyPrintResultCh(ch)
}
