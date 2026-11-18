package main

import (
	"fmt"
	"math/rand/v2"
)

type ILimiter interface {
	limiter([]int) []int
}

type NumberLimit struct{}
type SumLimit struct{}

func (nl NumberLimit) limiter() {

}

func main() {
	nums := NumbersGenerator()
}

func AcceptAll(nums []int, l ILimiter) []int {
	return l.limiter(nums)
}

func PrintSlice(text string, slice []int) {
	fmt.Print(text)
	for i := 0; i < len(slice); i++ {
		fmt.Print(slice[i], " ")
	}
	fmt.Print("\n")
}

func NumbersGenerator(sizeNumber, lenSlice int) []int {
	numbers := []int{}
	for i := 0; i < lenSlice; i++ {
		numbers = append(numbers, rand.IntN(sizeNumber))
	}
	return numbers
}
