package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"

	"github.com/drone/drone-plugin-go/plugin"
)

func main() {
	var repo = plugin.Repo{}
	var build = plugin.Build{}
	var vargs = struct {
		Urls []string `json:"urls"`
	}{}

	plugin.Param("repo", &repo)
	plugin.Param("build", &build)
	plugin.Param("vargs", &vargs)
	plugin.Parse()

	// data structure
	data := struct {
		Repo  plugin.Repo  `json:"repo"`
		Build plugin.Build `json:"build"`
	}{repo, build}

	// json payload that will be posted
	payload, err := json.Marshal(&data)
	if err != nil {
		os.Exit(1)
	}

	// post payload to each url
	for _, url := range vargs.Urls {
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			os.Exit(1)
		}
		resp.Body.Close()
	}
}
