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
	randomNumbers := make(chan int)
	squaringNumbers := make(chan int)

	go func(randomNumbers chan int) {
		defer close(randomNumbers)
		for i := 0; i < lengthOfArray; i++ {
			randomNumbers <- rand.IntN(sizeOfNumber)
		}
	}(randomNumbers)

	go func(squaringNumbers chan int) {
		defer close(squaringNumbers)
		for randNumber := range randomNumbers {
			squaringNumbers <- randNumber * randNumber
		}
	}(squaringNumbers)

	/*fmt.Print("Числа массива: ")
	for ranNum := range randomNumbers {
		fmt.Print(ranNum, " ")
	}
	fmt.Print("\nКвадраты массива: ")
	for squarNum := range squaringNumbers {
		fmt.Print(squarNum, " ", 1)
	}*/
	fmt.Print("\n\n")
}

func GenerateArrayNumbers(randomNumbers chan int, lengthOfArray int, sizeOfNumber int) {
	defer close(randomNumbers)
	for i := 0; i < lengthOfArray; i++ {
		randomNumbers <- rand.IntN(sizeOfNumber)
	}
}

func SquaringNumbers(randomNumbers <-chan int, squaringNumbers chan int) {
	for randNumber := range randomNumbers {
		squaringNumbers <- randNumber * randNumber
	}
	close(squaringNumbers)
}
