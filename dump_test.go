package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDumpData(t *testing.T) {
	// String
	output := &bytes.Buffer{}
	dumpData(output, "TEST 1", "test_string")
	assert.Equal(t, "\n---START TEST 1---\n\"test_string\"\n---END TEST 1---\n", output.String())

	// JSON encoding
	output.Reset()
	dumpData(output, "TEST 2", map[string]int{"one": 1})
	assert.Equal(t, "\n---START TEST 2---\n{\n\t\"one\": 1\n}\n---END TEST 2---\n", output.String())
}

func TestDumpFile(t *testing.T) {
	// Write to a new file
	const path = "/tmp/drone-gke-test-dump"
	data := []byte("hello, gke")
	err := ioutil.WriteFile(path, data, 0644)

	if assert.NoError(t, err) {
		output := &bytes.Buffer{}
		dumpFile(output, "TEST FILE", path)
		assert.Equal(t, "\n---START TEST FILE---\nhello, gke\n---END TEST FILE---\n", output.String())
	}

	// Delete file
	err = os.Remove(path)
	assert.NoError(t, err)
}
