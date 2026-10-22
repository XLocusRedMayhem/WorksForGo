package main

import (
	"fmt"
	"math/rand/v2"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	lengthOfArray := 0
	sizeOfNumber := 0
	fmt.Print("\nВведите размер числового массива: ")
	fmt.Scan(&lengthOfArray)
	fmt.Print("Введите размер генерируемых чисел: ")
	fmt.Scan(&sizeOfNumber)

	wg.Add(1)
	go GenerataArrayNumbers(&wg, lengthOfArray, sizeOfNumber)
	wg.Wait()

	fmt.Print("\n\n")
}

func GenerataArrayNumbers(wg *sync.WaitGroup, lengthOfArray int, sizeOfNumber int) {
	defer wg.Done()
	numbers := make([]int, lengthOfArray)
	for i := range lengthOfArray {
		numbers[i] = rand.IntN(sizeOfNumber)
	}
	fmt.Print("Числа массива: ")
	for _, number := range numbers {
		fmt.Print(number, " ")
	}
}
