package main

import (
	"fmt"
)

// Даны два числа. Определить цифры, входящие в запись как первого, так и второго числа
// Программа получает на вход два числа. Гарантируется, что цифры в числах не повторяются.
// Числа в пределах от 0 до 10000
// Программа должна вывести цифры, которые имеются в обоих числах, через пробел. Цифры выводятся в
// порядке их нахождения в первом числе.
func main() {
	var num1 string
	fmt.Print("Введите первое число: ")
	fmt.Scanln(&num1)
	var num2 string
	fmt.Print("Введите второе число: ")
	fmt.Scanln(&num2)
	fmt.Println("Ответ:", Similarity(num1, num2))
}
func Reduction(num string) string {
	res := ""
	for i := 0; i < len(num); i++ {
		for j := 0; j < len(num); j++ {
			if i == j {
				res += string(num[i])
			} else if string(num[i]) != string(num[j]) {
				continue
			} else {
				break
			}
		}
	}
	return res
}
func AddingSpaces(res string) string{
	res := 
}
func Similarity(num1, num2 string) string {
	res:= ""
	for i := 0; i < len(num1); i++ {
		for j := 0; j < len(num2); j++ {
			if string(num1[i]) == string(num2[j]) {
				res += string(num1[i])
			}
		}
	}
	return AddingSpaces(Reduction(res))
}
