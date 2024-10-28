package impl

import (
	fm "fmt"
)

func withColor(color string, text string) string {
	return "\033[3" + color + "m" + text + "\033[0m"
}

func Red(text string) string {
	return withColor("1", text)
}

func Green(text string) string {
	return withColor("2", text)
}

func Yellow(text string) string {
	return withColor("3", text)
}

func Blue(text string) string {
	return withColor("4", text)
}

func Purple(text string) string {
	return withColor("5", text)
}

func PrintList(title string, empty string, data []string) {
	if len(data) == 0 {
		fm.Println(empty)
		return
	}
	fm.Println(title)
	for i, line := range data {
		fm.Printf("  %d - %s\n", i, line)
	}
}
