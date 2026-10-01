package main

import "fmt"

// По данному (любому) трехзначному числу определите, все ли его цифры различны
// На вход подается одно натуральное трехзначное число
// Выведите "YES", если все цифры числа различны, в противном случае - "NO".
func main() {
	var num int
	fmt.Scanln(&num)
	var (
		num0 = num / 100
		num1 = num % 100 / 10
		num2 = num % 10
	)
	if num0 == num1 || num0 == num2 || num1 == num2 {
		fmt.Println("NO")
	} else {
		fmt.Println("YES")
	}
}
