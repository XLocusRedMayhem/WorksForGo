package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var numbers []int
	for i := range 15 {
		numbers = append(numbers, i)
	}
	numbers = append(numbers, 536)
	for i := range 15 {
		numbers = append(numbers, i+16)
	}

	fmt.Print("Числа среза: ")
	for i := 0; i < len(numbers); i++ {
		fmt.Print(numbers[i], " ")
	}
	fmt.Print("\n")

	ch := make(chan int, 2)

	go func() {
		for _, number := range numbers {
			ch <- number
		}
		close(ch)
	}()

	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		nameGorutine := fmt.Sprint("Горутина ", i+1)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					fmt.Print(nameGorutine, " завершила работу\n")
					return
				case value, endReading := <-ch:
					if !endReading {
						return
					}
					if value > 50 {
						fmt.Print(nameGorutine, " завершила работу число привысило 50\n")
						cancel()
						return
					}
					select {
					case <-ctx.Done():
						fmt.Print(nameGorutine, " завершила работу\n")
						return
					default:
						fmt.Print(nameGorutine, " начала обработку числа ", value, "\n")
						time.Sleep(2 * time.Second)
						fmt.Print(nameGorutine, " завершила обработку числа ", value, "\n")
					}
				}
			}
		}()
	}
	wg.Wait()
}
