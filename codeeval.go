package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
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
	defer closeNoError(rsp.Body)
	d, err = ioutil.ReadAll(rsp.Body)
	must(err)
	if rsp.StatusCode != 200 {
		err = fmt.Errorf("evalCode: request failed with '%s'", rsp.Status)
		if len(d) > 0 {
			logf(ctx(), "\nServer error:\n%s\n", string(d))
		}
		req, _ := json.MarshalIndent(e, "", "  ")
		logf(ctx(), "evalCode: request:\n%s\n", string(req))
		dbgEval(e)
		return nil, err
	}
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
		logf(ctx(), "File: '%s', Content:\n%s\n", name, s)
	}

	if e.Command != "" {
		logf(ctx(), "Command: '%s'\n", e.Command)
	}
}

func dbgEvalResponse(r *EvalResponse) {
	logf(ctx(), "\nEvalResponse:\n")
	if r.Stdout != "" {
		logf(ctx(), "Stdout:\n%s\n", r.Stdout)
	}
	if r.Stderr != "" {
		logf(ctx(), "Stderr:\n%s\n", r.Stderr)
	}
	if r.Error != "" {
		logf(ctx(), "Error:\n%s\n", r.Error)
	}
	logf(ctx(), "DurationMS: %.2f\n\n", r.DurationMS)
}
