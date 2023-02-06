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

func setupCerts() error {
	// copy the ca certificate
	// TODO: non-platform specific
	certs, err := ioutil.ReadFile("/etc/ssl/certs/ca-certificates.crt")
	if err != nil {
		return fmt.Errorf("reading existing certs: %w", err)
	}
	certLines := splitLines(string(certs))

	rootCa, err := ioutil.ReadFile("/customcerts/ca/rootCA.pem")
	if err != nil {
		return fmt.Errorf("reading mitm cert: %w", err)
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
		return fmt.Errorf("creating certificate file %w", err)
	}
	if _, err := f.WriteString(strings.Join(allLines, "\n")); err != nil {
		return fmt.Errorf("writing certificate contents: %w", err)
	}
	fmt.Println("certificates updated")

	return nil
}

func run() error {
	fmt.Println("Init process")
	if err := setupCerts(); err != nil {
		return fmt.Errorf("setting up certificates: %w", err)
	}

	time.Sleep(86400 * time.Second)
	return nil
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
