package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

func SliceGenerator() []int {
	numbers := []int{}
	for i := 0; i < 30; i++ {
		if i == 15 {
			numbers = append(numbers, 77)
		} else {
			numbers = append(numbers, i+1)
		}
	}
	PrintSlice("Числа среза: ", numbers)
	return numbers
}

func PrintSlice(text string, numbers []int) {
	fmt.Print(text)
	for i := range numbers {
		fmt.Print(numbers[i], " ")
	}
	fmt.Println()
}

func GeneratorCh(numbers []int) chan int {
	ch := make(chan int)
	go func() {
		for i := range numbers {
			ch <- numbers[i]
		}
		close(ch)
	}()
	return ch
}

func Workers(ctx context.Context, cansel context.CancelFunc, ch chan int, numWorkers int) {
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go Worker(&wg, ctx, cansel, ch, i)
	}
	wg.Wait()
}

func Worker(wg *sync.WaitGroup, ctx context.Context, cansel context.CancelFunc, ch chan int, id int) {
	defer wg.Done()
	fmt.Print("Горутина ", id, " начала работу")
	for {
		select {
		case number, ok := <-ch:
			if !ok {
				fmt.Print("Горутина ", id, " завершила работу")
				return
			}
			if number > 50 {
				fmt.Print("Горутина ", id, "принудительно завершила работку, ошибка")
				cansel()
				return
			}
			Process(id, number)
		case <-ctx.Done():
			fmt.Print("Горутина ", id, "принудительно завершила работку, ошибка")
			return
		}
	}
}

func Process(id int, number int) {
	fmt.Print("Горутина ", id, ": начала обрабатку числа ", number, "\n")
	time.Sleep(3 * time.Second)
	fmt.Print("Горутина ", id, ": завершила обработку числа ", number, "\n")
}

func main() {
	numWorkers := 3
	ctx, cansel := context.WithCancel(context.Background())
	defer cansel()
	ch := GeneratorCh(SliceGenerator())
	Workers(ctx, cansel, ch, numWorkers)
}
