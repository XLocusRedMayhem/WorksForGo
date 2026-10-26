package main

func main() { // не работает
	var numbers []int = []int{53, 23, 34, 26, 45, 34}
	resultsСhannel := make(chan int)
	go func() {
		defer close(resultsСhannel)
		for _, number := range numbers {
			resultsСhannel <- number
		}
	}()
}
