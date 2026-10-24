package main

import (
	"fmt"
	"math/rand/v2"
)

func main() {

	lengthOfArray := 0
	sizeOfNumber := 0
	fmt.Print("\nВведите размер числового массива: ")
	fmt.Scan(&lengthOfArray)
	fmt.Print("Введите размер генерируемых чисел: ")
	fmt.Scan(&sizeOfNumber)
	generated := make(chan int)
	squares := make(chan int)
	go func(generated chan int) {
		for range lengthOfArray {
			generated <- rand.IntN(sizeOfNumber)
		}
	}()
	go func() {
		number := <-generated
		squares <- number * number
	}()

	var numbers []int
	for number := range squares {
		numbers = append(numbers, number)
		fmt.Println(number)
	}
	close(generated)
	close(squares)
	fmt.Print("\n\n")
}
