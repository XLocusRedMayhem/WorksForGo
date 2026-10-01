package main

import "fmt"

// Вклад в банке составляет x рублей. Ежегодно он увеличивается на p процентов, после чего дробная часть
// копеек отбрасывается. Каждый год сумма вклада становится больше. Определите, через сколько лет вклад
// составит не менее y Программа получает на вход три натуральных числа: x, p, y Программа должна
// вывести одно целое число.
func main() {
	var contribution, percent, totalAmount int
	years := 0
	fmt.Print("Введите вклад, процент, итоговую сумму через пробел: ")
	fmt.Scan(&contribution, &percent, &totalAmount)
	for contribution < totalAmount {
		contribution += contribution / percent
		years++
	}
	fmt.Println(years, "лет потребуется чтобы накописть", totalAmount, "рублей.")
}
