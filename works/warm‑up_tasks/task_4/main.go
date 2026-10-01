package main

import "fmt"

// Напишите программу, которая в последовательности чисел находит сумму
// двузначных чисел, кратных 8. Программа в первой строке получает на
// вход число n - количество чисел в последовательности, во второй
// строке -- n чисел, входящих в данную последовательность.
func main() {
	var amountNumber int
	var number int
	var sumNambers int = 0
	fmt.Print("Количество чисел: ")
	fmt.Scan(&amountNumber)
	if amountNumber <= 0 {
		fmt.Println("Количетсво чисел не может быть 0 или отрицательным")
	} else {
		fmt.Print("Введите числа через пробел: ")
		for i := 0; i < amountNumber; i++ {
			fmt.Scan(&number)
			if number < 10 && number > 99 {
				i--
				number = 0
			} else {
				if number%8 == 0 {
					sumNambers += number
				}
			}
		}
		fmt.Println("Сумма чисел:", sumNambers)
	}
}
