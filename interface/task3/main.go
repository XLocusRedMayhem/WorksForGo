package main

import (
	"fmt"
	"math/rand/v2"
)

type ISource interface {
	source() string
}

type ReadingLines struct {
	strs    []string
	counter int
}
type StringsGeneration struct {
	strs    []string
	counter int
}

func (rl *ReadingLines) source() string {
	rl.counter++
	if rl.counter-1 > len(rl.strs)-1 {
		return ""
	} else {
		return rl.strs[rl.counter-1]
	}
}

func (sg *StringsGeneration) source() string {
	sg.counter++
	if sg.counter-1 > len(sg.strs)-1 {
		return ""
	} else {
		return sg.strs[sg.counter-1]
	}
}

func main() {
	readingLines := ReadingLines{strs: []string{"Привет у-у-вася, как дела?", "Всё окей, как у тебя?", "Всё норм пельмешки хаваю"}, counter: 0}
	stringsGenerator := StringsGeneration{strs: StringsGenerator(10, 4), counter: 0}

	strs := CollectAll(&readingLines)
	PrintSlice("Строки: ", strs)

	strs = CollectAll(&stringsGenerator)
	PrintSlice("Строки: ", strs)
}

func CollectAll(s ISource) []string {
	strs := []string{}
	for {
		str := s.source()
		if str != "" {
			strs = append(strs, str)
		} else {
			break
		}
	}
	return strs
}

func PrintSlice(text string, slice []string) {
	fmt.Print(text)
	for i := 0; i < len(slice); i++ {
		fmt.Print(slice[i], " ")
	}
	fmt.Print("\n")
}

func StringsGenerator(lenWord, lenSlice int) []string {
	strs := []string{}
	for i := 0; i < lenSlice; i++ {
		strs = append(strs, StringGenerator(lenWord))
	}
	return strs
}

func StringGenerator(lenWord int) string {
	str := ""
	for i := 0; i < lenWord; i++ {
		randNumber := rand.Int32N(123)
		// 48 - 57, 65 - 90, 97 - 122
		if randNumber >= 48 && randNumber <= 57 {
			str += string(rune(randNumber))
		} else if randNumber >= 65 && randNumber <= 90 {
			str += string(rune(randNumber))
		} else if randNumber >= 97 && randNumber <= 122 {
			str += string(rune(randNumber))
		} else {
			i--
			continue
		}
	}
	return str
}
