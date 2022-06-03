package webhook

import (
	"encoding/json"
	"fmt"
	"gopkg.in/go-playground/webhooks.v3/github"
	githubhook "gopkg.in/rjz/githubhook.v0"
	"net/http"
	"newproj/ci"
	"os"
	"path/filepath"
)

func GithubWebhookHandler(req *http.Request) {
	secret := []byte(os.Getenv("GITHUB_WEBHOOK_SECRET"))
	hook, err := githubhook.Parse(secret, req)
	if err != nil {
		fmt.Println("Error parsing webhook", err)
		return
	}

	fmt.Println("Received a ", hook.Event, "event")

	var job *ci.JobDetails

	switch hook.Event {
	case "ping":
		job, err = buildPingEventJob(hook.Payload)
	case string(github.PushEvent):
		job, err = buildPushEventJob(hook.Payload)
	}
	//case string(github.PullRequestEvent):
	//	job, err = buildPREventJob(hook.Payload)
	//}

	if err != nil {
		fmt.Printf("Build job error for %s event. Error: %s\n", hook.Event, err)
		return
	}

	ci.Run(job)
}

// ManualTrigger manually triggers the ci job
func ManualTrigger(repo, owner, sha, language, revert, url string, updateBuildStatusFunc func(string)) {
	job := &ci.JobDetails{
		LogFileName:            fmt.Sprintf("%s/%s/%s", owner, repo, sha),
		LogDirPath:             fmt.Sprintf("%s/%s", owner, repo),
		ProjectBranch:          sha,
		ProjectRepositoryURL:   url,
		ProjectLanguage:        language,
		ProjectRespositoryName: repo,
		IsRevert:               revert,
		UpdateBuildStatus:      updateBuildStatusFunc,
	}

	fmt.Println("Here's the job details: ", job)
	ci.Run(job)
}

func buildPushEventJob(payload []byte) (*ci.JobDetails, error) {
	evt := github.PushPayload{}
	if err := json.Unmarshal(payload, &evt); err != nil {
		return nil, err
	}

	branch := evt.After
	job := &ci.JobDetails{
		LogFileName:            filepath.Join(evt.Repository.FullName, branch),
		LogDirPath:             filepath.Join(evt.Repository.FullName),
		ProjectBranch:          branch,
		ProjectRepositoryURL:   evt.Repository.SSHURL,
		ProjectLanguage:        *evt.Repository.Language,
		ProjectRespositoryName: evt.Repository.Name,
	}
	return job, nil
}

//func buildPREventJob(payload []byte) (*ci.JobDetails, error) {
//	evt := github.PullRequestPayload{}
//	if err := json.Unmarshal(payload, &evt); err != nil {
//		return nil, err
//	}
//
//	branch := evt.PullRequest.Head.Sha
//	job := &ci.JobDetails{
//		LogFileName:            filepath.Join(evt.Repository.FullName, branch),
//		LogDirPath:             filepath.Join(evt.Repository.FullName),
//		ProjectBranch:          branch,
//		ProjectRepositoryURL:   evt.Repository.SSHURL,
//		ProjectLanguage:        *evt.Repository.Language,
//		ProjectRespositoryName: evt.Repository.Name,
//	}
//	return job, nil
//}

func buildPingEventJob(payload []byte) (*ci.JobDetails, error) {
	evt := github.WatchPayload{}
	if err := json.Unmarshal(payload, &evt); err != nil {
		return nil, err
	}
	branch := "master"
	job := &ci.JobDetails{
		LogFileName:            filepath.Join(evt.Repository.FullName, branch),
		LogDirPath:             filepath.Join(evt.Repository.FullName),
		ProjectBranch:          branch,
		ProjectRepositoryURL:   evt.Repository.SSHURL,
		ProjectLanguage:        *evt.Repository.Language,
		ProjectRespositoryName: evt.Repository.Name,
	}
	return job, nil
}
