package main

import (
	"bufio"
	"errors"
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

func isFile(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !s.IsDir()
}

var cannotFindCertFilename = errors.New("cannot find certificate bundle")

// findCertFilename searches the filesystem for common paths to the certificate
// bundle. If it cannot find one, it returns an error.
func findCertFilename() (string, error) {
	candidates := []string{
		"/etc/ssl/certs/ca-certificates.crt",
		"/etc/ssl/certs/ca-bundle.crt",
	}
	for _, c := range candidates {
		if isFile(c) {
			return c, nil
		}
	}

	return "", cannotFindCertFilename
}

func setupCerts() error {
	// copy the ca certificate
	certFilename, err := findCertFilename()
	certs, err := ioutil.ReadFile(certFilename)
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
	f, err := os.Create(certFilename)
	if err != nil {
		return fmt.Errorf("creating certificate file %w", err)
	}
	if _, err := f.WriteString(strings.Join(allLines, "\n")); err != nil {
		return fmt.Errorf("writing certificate contents: %w", err)
	}
	fmt.Println("certificates updated")

	return nil
}

func main() {
	fmt.Println("Init process")
	if err := setupCerts(); err != nil {
		fmt.Printf("setting up certificates: %v\n", err)
	}

	time.Sleep(86400 * time.Second)
}
