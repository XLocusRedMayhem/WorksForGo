package main

import (
	"fmt"
	"sync"
)

func main() {
	var lengthOfArray int
	fmt.Print("\n", "Укажите размер массива натуральных чисел: ")
	fmt.Scan(&lengthOfArray)
	numbers := make([]int, lengthOfArray)
	fmt.Print("Введите числа через пробел: ")
	for i, number := range numbers {
		fmt.Scan(&number)
		numbers[i] = number
	}
	fmt.Print("Числа массива: ")
	for _, number := range numbers {
		fmt.Print(number, " ")
	}
	var wg sync.WaitGroup
	results := make(chan int)
	wg.Add(2)
	go func(numbers []int) {
		defer wg.Done()
		results <- Sum(numbers[:len(numbers)/2])
	}(numbers)
	go func(numbers []int) {
		defer wg.Done()
		results <- Sum(numbers[len(numbers)/2:])
	}(numbers)

	go func() {
		wg.Wait()
		close(results)
	}()

	wg.Wait()
	close(results)
	fmt.Print("\n\n")
}

func Sum(numbers []int) int {
	return 10
}
