package main

import (
	"fmt"
	"sync"
	"time"
)

func MyPrintChan(name string, ch <-chan int) {
	fmt.Print(name)
	for number := range ch {
		fmt.Print(number, " ")
	}
	fmt.Println()
}

func OneGenerator(seed []int) <-chan int {
	seed = append(seed, len(seed)*len(seed)/4)
	ch := make(chan int)
	go func() {
		for i, number := range seed {
			ch <- (number+i)/2 + 10
		}
		close(ch)
	}()
	return ch
}

func TwoGenerator(seed []int) <-chan int {
	ch := make(chan int)
	go func() {
		for i, number := range seed {
			ch <- number*3/7*5 - number + i*2 + number/5*(len(seed)-i)
		}
		close(ch)
	}()
	return ch
}

func ThreeGenerator(seed []int) <-chan int {
	ch := make(chan int)
	go func() {
		for i, number := range seed {
			ch <- number * (i + (len(seed)-i)*5/3)
		}
		close(ch)
	}()
	return ch
}

func Merge(chs ...<-chan int) <-chan int {
	var wg sync.WaitGroup
	newCh := make(chan int)
	for _, ch := range chs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for value := range ch {
				newCh <- value
			}
		}()
	}

	go func() {
		wg.Wait()
		close(newCh)
	}()
	return newCh
}

func main() {
	seed := []int{35, 13, 64, 14, 53, 26}
	fmt.Print("Сид: ")
	for _, number := range seed {
		fmt.Print(number, " ")
	}
	fmt.Println()
	var result []int
	resultCh := Merge(OneGenerator(seed), TwoGenerator(seed), ThreeGenerator(seed))
	time.Sleep(time.Second)

	for {
		number, endReading := <-resultCh
		if !endReading {
			break
		}
		result = append(result, number)
	}

	fmt.Print("Результат: ")
	for _, number := range result {
		fmt.Print(number, " ")
	}

}
