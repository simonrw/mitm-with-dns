package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func splitLines(s string) []string {
	var lines []string

	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}

	return lines
}

func main() {
	fmt.Println("Init process")

	// copy the ca certificate
	// TODO: non-platform specific
	certs, err := ioutil.ReadFile("/etc/ssl/certs/ca-certificates.crt")
	if err != nil {
		panic(err)
	}
	certLines := splitLines(string(certs))

	rootCa, err := ioutil.ReadFile("/customcerts/ca/rootCA.pem")
	if err != nil {
		panic(err)
	}
	caLines := splitLines(string(rootCa))

	var allLines []string
	for _, line := range certLines {
		allLines = append(allLines, line)
	}
	for _, line := range caLines {
		allLines = append(allLines, line)
	}

	// clobber output file
	f, err := os.Create("/etc/ssl/certs/ca-certificates.crt")
	if err != nil {
		panic(err)
	}
	if _, err := f.WriteString(strings.Join(allLines, "\n")); err != nil {
		panic(err)
	}
	fmt.Println("certificates updated")

	time.Sleep(86400 * time.Second)
}
