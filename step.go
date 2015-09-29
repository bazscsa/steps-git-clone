package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// -----------------------
// --- functions
// -----------------------

func validateRequiredInput(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("[!] Missing required input: %s", key)
	}
	return value, nil
}

func genericIsPathExists(pth string) (os.FileInfo, bool, error) {
	if pth == "" {
		return nil, false, errors.New("No path provided")
	}
	fileInf, err := os.Stat(pth)
	if err == nil {
		return fileInf, true, nil
	}
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	return fileInf, false, err
}

func isPathExists(pth string) (bool, error) {
	_, isExists, err := genericIsPathExists(pth)
	return isExists, err
}

func envmanAdd(key, value string) error {
	args := []string{"add", "--key", key}

	cmd := exec.Command("envman", args...)
	cmd.Stdin = strings.NewReader(value)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func doGitInit() error {
	cmd := exec.Command("git", "init")
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func doGitAddRemote(repositoryURL string) error {
	cmd := exec.Command("git", "remote", "add", "origin", repositoryURL)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func doGitFetch(pullRequestID, gitCheckoutParam string) error {
	args := []string{"fetch"}
	if pullRequestID != "" {
		args = append(args, "origin", "pull/"+pullRequestID+"/merge:"+gitCheckoutParam)
	}

	cmd := exec.Command("git", args...)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func doGitCheckout(gitCheckoutParam string) error {
	cmd := exec.Command("git", "checkout", gitCheckoutParam)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func doGitSubmodelueUpdate() error {
	cmd := exec.Command("git", "submodule", "update", "--init", "--recursive")
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func getGitLog(formatParam string) (string, error) {
	outBuffer := bytes.Buffer{}
	errBuffer := bytes.Buffer{}

	cmd := exec.Command("git", "log", "-1", "--format", formatParam)
	cmd.Stdin = nil
	cmd.Stdout = io.Writer(&outBuffer)
	cmd.Stderr = io.Writer(&errBuffer)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git log failed, err: %s, details: %s", err, errBuffer.String())
	}
	return outBuffer.String(), nil

}

func doGitClone(cloneIntoDir, repositoryURL, pullRequestID, gitCheckoutParam string) error {
	gitCheckPath := path.Join(cloneIntoDir, ".git")
	if exist, err := isPathExists(gitCheckPath); err != nil {
		return fmt.Errorf("Failed to file path (%s), err: %s", gitCheckPath, err)
	} else if exist {
		return fmt.Errorf(".git folder already exists in the destination dir (%s)", gitCheckPath)
	}

	if err := os.MkdirAll(cloneIntoDir, 0777); err != nil {
		return fmt.Errorf("Failed to create the clone_destination_dir at: %s", cloneIntoDir)
	}

	if err := doGitInit(); err != nil {
		return fmt.Errorf("Could not init git repository, err: %s", cloneIntoDir)
	}

	if err := doGitAddRemote(repositoryURL); err != nil {
		return fmt.Errorf("Could not add remote, err: %s", err)
	}

	if err := doGitFetch(pullRequestID, gitCheckoutParam); err != nil {
		return fmt.Errorf("Could not fetch from repository, err: %s", err)
	}

	if gitCheckoutParam != "" {
		if err := doGitCheckout(gitCheckoutParam); err != nil {
			return fmt.Errorf("Could not do checkout (%s), err: %s", gitCheckoutParam, err)
		}

		if err := doGitSubmodelueUpdate(); err != nil {
			return fmt.Errorf("Could not fetch from submodule repositories, err: %s", err)
		}

		// git clone stats
		commitStats := map[string]string{}
		commitHashStr, err := getGitLog("%H")
		if err != nil {
			fmt.Println(err)
		}
		commitStats["GIT_CLONE_COMMIT_HASH"] = commitHashStr

		commitMsgSubjectStr, err := getGitLog("%s")
		if err != nil {
			fmt.Println(err)
		}
		commitStats["GIT_CLONE_COMMIT_MESSAGE_SUBJECT"] = commitMsgSubjectStr

		commitMsgBodyStr, err := getGitLog("%b")
		if err != nil {
			fmt.Println(err)
		}
		commitStats["GIT_CLONE_COMMIT_MESSAGE_BODY"] = commitMsgBodyStr

		commitAuthorNameStr, err := getGitLog("%an")
		if err != nil {
			fmt.Println(err)
		}
		commitStats["GIT_CLONE_COMMIT_AUTHOR_NAME"] = commitAuthorNameStr

		commitAuthorEmailStr, err := getGitLog("%ae")
		if err != nil {
			fmt.Println(err)
		}
		commitStats["GIT_CLONE_COMMIT_AUTHOR_EMAIL"] = commitAuthorEmailStr

		commitCommiterNameStr, err := getGitLog("%cn")
		if err != nil {
			fmt.Println(err)
		}
		commitStats["GIT_CLONE_COMMIT_COMMITER_NAME"] = commitCommiterNameStr

		commitCommiterEmailStr, err := getGitLog("%ce")
		if err != nil {
			fmt.Println(err)
		}
		commitStats["GIT_CLONE_COMMIT_COMMITER_EMAIL"] = commitCommiterEmailStr

		for key, value := range commitStats {
			if err := envmanAdd(key, value); err != nil {
				fmt.Printf("Faild to export ouput: (%s), err: %s\n", key, err)
			}
		}
	} else {
		fmt.Println(" [!] No checkout parameter (branch, tag, commit hash or pull-request ID) provided!")
	}

	return nil
}

// -----------------------
// --- main
// -----------------------

func main() {
	//
	// Required parameters
	repoURL, err := validateRequiredInput("repository_url")
	if err != nil {
		log.Fatalf("Input validation failed, err: %s", err)
	}
	cloneIntoDir, err := validateRequiredInput("clone_into_dir")
	if err != nil {
		log.Fatalf("Input validation failed, err: %s", err)
	}

	//
	// Optional parameters
	commit := os.Getenv("commit")
	tag := os.Getenv("tag")
	branch := os.Getenv("branch")
	pullRequestID := os.Getenv("pull_request_id")

	// Normalize input pathes
	absCloneIntoDir, err := filepath.Abs(cloneIntoDir)
	if err != nil {
		log.Fatalf("Failed to expand path (%s), err: %s", cloneIntoDir, err)
	}

	// Parse repo uri
	preparedRepoURL, err := url.Parse(repoURL)
	if err != nil {
		log.Fatalf("Failed to parse repo url (%s), err: %s", repoURL, err)
	}

	// do clone
	gitCheckoutParam := ""
	if len(pullRequestID) > 0 {
		gitCheckoutParam = "pull/" + pullRequestID
	} else if len(commit) > 0 {
		gitCheckoutParam = commit
	} else if len(tag) > 0 {
		// since git 1.8.x tags can be specified as "branch" too ( http://git-scm.com/docs/git-clone )
		//  [!] this will create a detached head, won't switch to a branch!
		gitCheckoutParam = tag
	} else if len(branch) > 0 {
		gitCheckoutParam = branch
	} else {
		fmt.Println(" [!] No checkout parameter found")
	}

	if err := doGitClone(absCloneIntoDir, preparedRepoURL.String(), pullRequestID, gitCheckoutParam); err != nil {
		log.Fatalf("git clone failed, err: %s", err)
	}
}
