package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/kjk/u"
)

const (
	githubServer = "https://api.github.com"
)

// https://developer.github.com/v3/gists/#create-a-gist
type GistCreate struct {
	Description string                  `json:"description"`
	Public      bool                    `json:"public"`
	Files       map[string]*GistNewFile `json:"files"`
}

type GistNewFile struct {
	Content string `json:"content"`
}

// https://developer.github.com/v3/gists/#get-a-single-gist
type Gist struct {
	// comes from the API
	URL         string               `json:"url"`
	ForksURL    string               `json:"forks_url"`
	CommitsURL  string               `json:"commits_url"`
	ID          string               `json:"id"`
	NodeID      string               `json:"node_id"`
	GitPullURL  string               `json:"git_pull_url"`
	GitPushURL  string               `json:"git_push_url"`
	HTMLURL     string               `json:"html_url"`
	Files       map[string]*GistFile `json:"files"`
	Public      bool                 `json:"public"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
	Description string               `json:"description"`
	Comments    int64                `json:"comments"`
	User        json.RawMessage      `json:"user"`
	CommentsURL string               `json:"comments_url"`
	Owner       *GitOwner            `json:"owner"`
	Forks       json.RawMessage      `json:"forks"`
	History     []*GistHistory       `json:"history"`
	Truncated   bool                 `json:"truncated"`

	// set by us
	Raw string `json:"-"`
}

type GistFile struct {
	Filename  string `json:"filename"`
	Type      string `json:"type"`
	Language  string `json:"language"`
	RawURL    string `json:"raw_url"`
	Size      int64  `json:"size"`
	Truncated bool   `json:"truncated"`
	Content   string `json:"content"`
}

type GistHistory struct {
	User         GitOwner        `json:"user"`
	Version      string          `json:"version"`
	CommittedAt  string          `json:"committed_at"`
	ChangeStatus GitChangeStatus `json:"change_status"`
	URL          string          `json:"url"`
}

type GitChangeStatus struct {
	Total     int64 `json:"total"`
	Additions int64 `json:"additions"`
	Deletions int64 `json:"deletions"`
}

type GitOwner struct {
	Login             string `json:"login"`
	ID                int64  `json:"id"`
	NodeID            string `json:"node_id"`
	AvatarURL         string `json:"avatar_url"`
	GravatarID        string `json:"gravatar_id"`
	URL               string `json:"url"`
	HTMLURL           string `json:"html_url"`
	FollowersURL      string `json:"followers_url"`
	FollowingURL      string `json:"following_url"`
	GistsURL          string `json:"gists_url"`
	StarredURL        string `json:"starred_url"`
	SubscriptionsURL  string `json:"subscriptions_url"`
	OrganizationsURL  string `json:"organizations_url"`
	ReposURL          string `json:"repos_url"`
	EventsURL         string `json:"events_url"`
	ReceivedEventsURL string `json:"received_events_url"`
	Type              string `json:"type"`
	SiteAdmin         bool   `json:"site_admin"`
}

func gistDecode(s string) *Gist {
	var res Gist
	err := json.Unmarshal([]byte(s), &res)
	if err != nil {
		logf("json:\n%s\n", s)
	}
	must(err)
	res.Raw = s
	return &res
}

var (
	didNotifyUsingToken bool
)

func setGitHubToken(req *http.Request) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return
	}
	if !didNotifyUsingToken {
		logf("GITHUB_TOKEN set, using it for GitHub API requests\n")
		didNotifyUsingToken = true
	}
	req.Header.Set("Authorization", "token "+token)
}

func githubGet(endpoint string) string {
	uri := githubServer + endpoint
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	must(err)
	setGitHubToken(req)
	resp, err := http.DefaultClient.Do(req)
	must(err)
	panicIf(resp.StatusCode >= 400, "http.Do('%s') failed with '%s'", uri, resp.Status)
	defer u.CloseNoError(resp.Body)
	d, err := ioutil.ReadAll(resp.Body)
	must(err)
	return string(d)
}

func githubPost(endpoint string, body io.Reader) string {
	uri := githubServer + endpoint
	req, err := http.NewRequest(http.MethodPost, uri, body)
	must(err)
	setGitHubToken(req)
	resp, err := http.DefaultClient.Do(req)
	must(err)
	panicIf(resp.StatusCode >= 400, "http.Do('%s') failed with '%s'", uri, resp.Status)
	defer u.CloseNoError(resp.Body)
	d, err := ioutil.ReadAll(resp.Body)
	must(err)
	return string(d)
}

func gistDownloadMust(gistID string) *Gist {
	endpoint := "/gists/" + gistID
	gistStr := githubGet(endpoint)
	gist := gistDecode(gistStr)
	// TODO: download truncated files
	panicIf(gist.Truncated)
	for _, f := range gist.Files {
		panicIf(f.Truncated)
	}
	return gist
}

func createGistMust(gist *GistCreate) *Gist {
	endpoint := "/gists"
	d, err := json.Marshal(gist)
	must(err)
	body := bytes.NewReader(d)
	gistStr := githubPost(endpoint, body)
	return gistDecode(gistStr)
}
