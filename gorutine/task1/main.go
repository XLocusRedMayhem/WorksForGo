package main

import (
	"fmt"
	"strconv"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	var num string
	fmt.Print("\n", "Напишите число: ")
	fmt.Scan(&num)
	wg.Add(2)
	go SumNum(num[:len(num)/2], &wg, 1)
	go SumNum(num[len(num)/2:], &wg, 2)
	wg.Wait()
	fmt.Print("\n")
}

func SumNum(num string, wg *sync.WaitGroup, number int) {
	defer wg.Done()
	var sum int
	fmt.Print(number, " срез: ", num, "\n")
	for i := 0; i < len(num); i++ {
		conNum, err := strconv.Atoi(num)
		fmt.Print(number, " ", conNum, "\n")
		if err != nil {
			fmt.Print("Ошибка", "\n")
		}
		sum += conNum
	}

	fmt.Print("сумма ", number, " среза: ", sum, "\n")
}
