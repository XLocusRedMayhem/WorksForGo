package main

import (
	"fmt"
	"sync"
)

type Task struct {
	ID     int
	Number int
}

type Result struct {
	Number int
}

func main() { // не работает
	var numbers []int = []int{53, 23, 34, 26, 45, 34}

	var wg sync.WaitGroup
	tasksСhannel := make(chan Task)
	resultsChannel := make(chan Result)

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go Worker(tasksСhannel, resultsChannel, i, &wg)
	}

	go func() {
		defer close(tasksСhannel)
		for id, number := range numbers {
			tasksСhannel <- Task{ID: id, Number: number}
		}
	}()

	go func() {
		wg.Wait()
		close(resultsChannel)
	}()

	for result := range resultsChannel {
		fmt.Print("Резултат: ")
		fmt.Print(result.Number, " ")
	}

}

func Worker(tasksChannel <-chan Task, resultsСhannel chan<- Result, id int, wg *sync.WaitGroup) {
	defer wg.Done()
	for task := range tasksChannel {
		resultsСhannel <- Result{Number: task.Number * 2}
	}
}
