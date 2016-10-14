package main

import (
	"github.com/foozlevazquez/golang-logwatcher"
	"bufio"
	"os"
	"fmt"
	"log"
)

func main() {
	var l = log.New(os.Stdin, "example", log.LstdFlags)
	lw := logwatcher.New(&logwatcher.Config{
		Filename: os.Args[1],
		Log: l,
	})
	var scanner = bufio.NewScanner(lw)

	lines := []string{}
	for {
		if scanner.Scan() {
			ln := scanner.Text()
			if err := scanner.Err(); err != nil {
				panic(err)
			}
			lines = append(lines, ln)
		} else {
			break
		}
	}
	fmt.Printf("%d\n", len(lines))
}
