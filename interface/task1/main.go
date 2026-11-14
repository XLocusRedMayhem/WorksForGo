package main

type Preducate interface {
	preducate
}

import (
	"fmt"
	"math/rand/v2"
)

func main() {
	maxNumber := 100
	numberOfNumbers := 60
	numbers := []int{}
	for i := 0; i < numberOfNumbers; i++ {
		numbers = append(numbers, rand.IntN(maxNumber)-rand.IntN(maxNumber))
	}
	PrintSlice("Числа среза: ", numbers)
}

func PrintSlice(text string, slice []int) {
	fmt.Print(text)
	for i := 0; i < len(slice)-1; i++ {
		fmt.Print(slice[i], " ")
	}
	fmt.Print("\n")
}

func Filter(numbers []int, p preducate) {

}
