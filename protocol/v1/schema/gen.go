// +build ignore

// go generate utility to encode the json schemas and embed them
// into a out file - typically schemas.go, works via something
// like `go run gen.go -- foo.go` to add a init func to foo.go

package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"time"
)

func writeSchema(schema string, variable string, outfile *os.File) {
	fname := fmt.Sprintf("schema/%s.json", schema)

	infile, err := ioutil.ReadFile(fname)
	if err != nil {
		log.Fatalf("Could not open %s: %s", fname, err)
	}

	encoded := base64.StdEncoding.EncodeToString(infile)

	outfile.WriteString(fmt.Sprintf("	%s, _ = base64.StdEncoding.DecodeString(`%s`)\n", variable, encoded))
}

func main() {
	if len(os.Args) <= 2 {
		log.Fatal("Please specify a file to edit")
	}

	fname := os.Args[len(os.Args)-1]

	fmt.Printf("Attempting to generate Base64 encoded JSON Schemas in %s...\n", fname)

	infile, err := os.Open(fname)
	if err != nil {
		log.Fatalf("Could not open %s: %s", fname, err)
	}
	defer infile.Close()

	tmpfile, err := ioutil.TempFile("", "generate")
	if err != nil {
		log.Fatalf("Could not open tempfile: %s", err)
	}
	defer os.Remove(tmpfile.Name())

	scanner := bufio.NewScanner(infile)

	found := false

	for scanner.Scan() {
		if regexp.MustCompile(`^func init\(\) {`).MatchString(scanner.Text()) {
			found = true
		}

		if !found {
			tmpfile.WriteString(scanner.Text() + "\n")
		}
	}

	tmpfile.WriteString("func init() {\n")
	tmpfile.WriteString(fmt.Sprintf("	// generated using gen.go at %s\n", time.Now()))
	tmpfile.WriteString("	schemas = jsonSchemas{}\n\n")

	writeSchema("reply", "schemas.ReplyV1", tmpfile)
	writeSchema("request", "schemas.RequestV1", tmpfile)
	writeSchema("secure_reply", "schemas.SecureReplyV1", tmpfile)
	writeSchema("secure_request", "schemas.SecureRequestV1", tmpfile)
	writeSchema("transport", "schemas.TransportV1", tmpfile)

	tmpfile.WriteString("}")

	infile.Close()

	os.Rename(tmpfile.Name(), infile.Name())

	fmt.Println("..done")
}
