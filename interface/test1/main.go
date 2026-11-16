package main

import (
	"fmt"
	"math/rand/v2"
)

type IPreducate interface {
	preducate(number int) bool
}

type Even struct{ contidion bool }

func (e Even) preducate(number int) bool {
	return number%2 == 0
}

func main() {
	maxNumber := 100
	numberOfNumbers := 60
	numbers := []int{}
	evenNumber := Even{}
	for i := 0; i < numberOfNumbers; i++ {
		numbers = append(numbers, rand.IntN(maxNumber)-rand.IntN(maxNumber))
	}
	PrintSlice("Числа среза: ", numbers)
	Filter(numbers, evenNumber)
}

func PrintSlice(text string, slice []int) {
	fmt.Print(text)
	for i := 0; i < len(slice)-1; i++ {
		fmt.Print(slice[i], " ")
	}
	fmt.Print("\n")
}

func Filter(numbers []int, p IPreducate) []int {
	newNumbers := []int{}
	for i := 0; i < len(numbers); i++ {
		if p.preducate(numbers[i]) {
			numbers = append(newNumbers, numbers[i])
		}
	}
	return newNumbers
}
