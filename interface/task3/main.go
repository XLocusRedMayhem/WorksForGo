package main

import (
	"fmt"
)

type ISource interface {
	source() string
}

type ReadingLines struct {
	strs    []string
	counter int
}
type StringGeneration struct{ strs []string }

func (rl ReadingLines) source() string {
	if rl.counter >= len(rl.strs) {
		return ""
	} else {
		rl.counter++
		return rl.strs[rl.counter]
	}
}
func (sg StringGeneration) source() string {

	return "str"
}

func main() {
	//readingLines := ReadingLines{strs: []string{"Привет у-у-вася, как дела?", "Всё окей как у тебя", "Всё норм пельмешки хаваю"}, counter: 0}

	//strs := collectAll(readingLines)

	fmt.Print()
}

func PrintSlice(text string, slice []string) {
	fmt.Print(text)
	for i := 0; i < len(slice); i++ {
		fmt.Print(slice[i], " ")
	}
	fmt.Print("\n")
}

func StringGenerator() []string {
	return []string{}
}

func collectAll(s ISource) []string {
	strs := []string{}
	for {
		if s.source() != "" {
			strs = append(strs, s.source())
		} else {
			break
		}
	}
	return strs
}
