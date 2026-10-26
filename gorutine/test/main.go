package main

import (
	"fmt"
	"math/rand/v2"
	"sync"
)

func main() {

	lengthOfArray := 6
	sizeOfNumber := 100
	/*fmt.Print("\nВведите размер числового массива: ")
	fmt.Scan(&lengthOfArray)
	fmt.Print("Введите размер генерируемых чисел: ")
	fmt.Scan(&sizeOfNumber)*/
	generated := make(chan int)
	squares := make(chan int)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(generated)
		for i := 0; i < lengthOfArray; i++ {
			generated <- rand.IntN(sizeOfNumber)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(squares)
		for i := range generated {
			squares <- i * i
		}
	}()

	for result := range squares {

	}

	go func() {
		wg.Wait()
	}()
	var a string
	fmt.Scan(&a)
	fmt.Print("\n\n")
}
