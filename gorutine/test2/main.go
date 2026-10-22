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
	fmt.Print("\n")
	var wg sync.WaitGroup
	results := make(chan int)
	wg.Add(1)
	go Sum(&wg, numbers[:len(numbers)/2], results)
	wg.Add(1)
	go Sum(&wg, numbers[len(numbers)/2:], results)
	go func() {
		wg.Wait()
		close(results)
	}()
	result := 0
	for slices := range results {
		result += slices
	}
	fmt.Print("Сумма двух сум срезов: ", result)
	fmt.Print("\n\n")
}
func Sum(wg *sync.WaitGroup, numbers []int, results chan<- int) {
	defer wg.Done()
	sum := 0
	fmt.Print("Срез массива чисел: ")
	for _, number := range numbers {
		fmt.Print(number, " ")
		sum += number
	}
	fmt.Print("\n")
	results <- sum
}
