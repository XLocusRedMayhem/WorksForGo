package main

import (
	"fmt"
	"math"
	"math/rand/v2"
)

type IMapping interface {
	mapping(number int) int
}

type MultiplyByTow struct{}
type AbsolutValue struct{}

func (mbt MultiplyByTow) mapping(number int) int {
	return number * 2
}
func (av AbsolutValue) mapping(number int) int {
	return int(math.Abs(float64(number)))
}

func main() {
	multiplyByTow := MultiplyByTow{}
	absolutValue := AbsolutValue{}

	maxNumber := 100
	numberOfNumbers := 10
	numbers := NumberGenerator(maxNumber, numberOfNumbers)

	PrintSlice("Числа среза            | ", numbers)
	fmt.Print("Изменённые срезы       |\n")
	PrintSlice("Число помножинные на 2 | ", Transform(numbers, multiplyByTow))
	PrintSlice("Модуль чисел           | ", Transform(numbers, absolutValue))
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

func Transform(numbers []int, m IMapping) []int {
	newNumbers := []int{}
	for i := 0; i < len(numbers); i++ {
		newNumbers = append(newNumbers, m.mapping(numbers[i]))
	}
	return newNumbers
}
