package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
)

func dumpData(w io.Writer, caption string, data interface{}) {
	fmt.Fprintf(w, "\n---START %s---\n", caption)
	defer fmt.Fprintf(w, "---END %s---\n", caption)

	b, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		fmt.Fprintf(w, "error marshalling: %s\n", err)
		return
	}

	w.Write(b)
	fmt.Fprintf(w, "\n")
}

func dumpFile(w io.Writer, caption, path string) {
	fmt.Fprintf(w, "\n---START %s---\n", caption)
	defer fmt.Fprintf(w, "---END %s---\n", caption)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Fprintf(w, "error reading file: %s\n", err)
		return
	}

	fmt.Fprintln(w, string(data))
}
