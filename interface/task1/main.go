package main

import (
	"fmt"
	"math/rand/v2"
)

type IPreducate interface {
	preducate(number int) bool
}

type Even struct{}
type Positive struct{}

func (e Even) preducate(number int) bool {
	return number%2 == 0
}
func (e Positive) preducate(number int) bool {
	return number > 0
}

func main() {
	evenNumber := Even{}
	positiveNumber := Positive{}

	maxNumber := 100
	numberOfNumbers := 60
	numbers := NumberGenerator(maxNumber, numberOfNumbers)

	PrintSlice("Числа среза: ", numbers)
	fmt.Print("Отфильтрованные срезы\n")
	PrintSlice("Целые числа: ", Filter(numbers, evenNumber))
	PrintSlice("Положительные числа: ", Filter(numbers, positiveNumber))
}

func PrintSlice(text string, slice []int) {
	fmt.Print(text)
	for i := 0; i < len(slice); i++ {
		fmt.Print(slice[i], " ")
	}
	fmt.Print("\n")
}

func NumberGenerator(maxNumber int, numberOfNumbers int) []int {
	numbers := []int{}
	for i := 0; i < numberOfNumbers; i++ {
		numbers = append(numbers, rand.IntN(maxNumber)-rand.IntN(maxNumber))
	}
	return numbers
}

func Filter(numbers []int, p IPreducate) []int {
	newNumbers := []int{}
	for i := 0; i < len(numbers); i++ {
		if p.preducate(numbers[i]) {
			newNumbers = append(newNumbers, numbers[i])
		}
	}
	return newNumbers
}
