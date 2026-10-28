package main

import (
	"fmt"
	"math/rand/v2"
	"sync"
)

func main() {
	lengthOfArray := 1
	sizeOfNumber := 1
	fmt.Print("\nВведите размер числового массива: ")
	fmt.Scan(&lengthOfArray)
	fmt.Print("Введите размер генерируемых чисел: ")
	fmt.Scan(&sizeOfNumber)
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

	var results []int
	for result := range squares {
		results = append(results, result)
	}

	go func() {
		wg.Wait()
	}()

	fmt.Print("Результат: ")
	for _, result := range results {
		fmt.Print(result, " ")
	}

	fmt.Print("\n\n")
}
