package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v41/github"
	"golang.org/x/oauth2"
)

var (
	gitDir = filepath.Join(os.Getenv("HOME"), "git")
)

func main() {
	token := os.Getenv("CLONE_GITHUB_TOKEN")
	if token == "" {
		panic("No CLONE_GITHUB_TOKEN")
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	ctx := context.Background()
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	noErr(os.Chdir(gitDir))

	urlChan := make(chan string, 50)
	go func() {
		err := getRepoURLs(client, urlChan, ctx)
		noErr(err)
	}()

	for gitUrl := range urlChan {
		dirName := strings.TrimSuffix(path.Base(gitUrl), ".git")
		if _, err := os.Stat(dirName); err != nil {
			fmt.Println(gitUrl, "->", dirName)
			cmd := exec.Command("git", "clone", gitUrl)
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout

			// Do not exit if a single clone fails - try others.
			// Errors will have been logged by the git command.
			_ = cmd.Run()
			fmt.Println()
		}
	}
}

func getRepoURLs(client *github.Client, urls chan<- string, ctx context.Context) error {
	opts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{
			Page:    0,
			PerPage: 25,
		},
	}

	defer close(urls)

	for {
		repos, resp, err := client.Repositories.List(ctx, "", opts)
		if err != nil {
			return err
		}

		for _, rep := range repos {
			urls <- rep.GetSSHURL()
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return nil
}

func noErr(err error) {
	if err != nil {
		panic(err)
	}
}
