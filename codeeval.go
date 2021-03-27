package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/kjk/u"
)

var (
	evalCodeServer = "https://codeeval.dev"
)

type File struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type Eval struct {
	Files    []File `json:"files"`
	Language string `json:"lang"`
	Command  string `json:"command,omitempty"`
	Stdin    string `json:"stdin,omitempty"`
}

type EvalResponse struct {
	Stdout     string  `json:"stdout"`
	Stderr     string  `json:"stderr"`
	Error      string  `json:"error,omitempty"`
	DurationMS float64 `json:"durationms"`
}

// http://localhost:8533

func evalCode(e *Eval) (*EvalResponse, error) {
	d, err := json.Marshal(e)
	must(err)
	uri := evalCodeServer + "/api/eval"
	body := bytes.NewReader(d)
	req, err := http.NewRequest("POST", uri, body)
	if err != nil {
		return nil, err
	}
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if rsp.StatusCode != 200 {
		err = fmt.Errorf("request failed with '%s'", rsp.Status)
		return nil, err
	}
	defer u.CloseNoError(rsp.Body)
	d, err = ioutil.ReadAll(rsp.Body)
	must(err)
	var res EvalResponse
	err = json.Unmarshal(d, &res)
	must(err)
	return &res, nil
}

func getFirstLines(s string, n int) string {
	lines := dataToLines([]byte(s))
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

func dbgEval(e *Eval) {
	for _, f := range e.Files {
		name := "no name"
		if f.Name != "" {
			name = f.Name
		}
		//s := getFirstLines(f.Content, 5)
		s := f.Content
		logf("File: '%s', Content:\n%s\n", name, s)
	}

	if e.Command != "" {
		logf("Command: '%s'\n", e.Command)
	}
}

func dbgEvalResponse(r *EvalResponse) {
	logf("\nEvalResponse:\n")
	if r.Stdout != "" {
		logf("Stdout:\n%s\n", r.Stdout)
	}
	if r.Stderr != "" {
		logf("Stderr:\n%s\n", r.Stderr)
	}
	if r.Error != "" {
		logf("Error:\n%s\n", r.Error)
	}
	logf("DurationMS: %.2f\n\n", r.DurationMS)
}
