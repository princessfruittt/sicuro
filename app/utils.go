package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/github"
	"github.com/gorilla/sessions"
	"newproj/app/vcs"
	"newproj/ci"
)

type projectLogListing struct {
	Name   string
	Active bool
}

type repoWithSubscriptionInfo struct {
	IsSubscribed bool
	*github.Repository
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := Templates.ExecuteTemplate(w, tmpl+".tmpl", data)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func logFilePathFromRequest(prefix string, r *http.Request) string {
	path, _ := filepath.Rel(prefix, r.URL.Path)
	return fmt.Sprintf("%s%s", filepath.Join(ci.LogDIR, path), ci.LogFileExt)
}

func fetchTemplates() (templates []string) {
	templateFolderName := "templates"
	templateFolder := filepath.Join(AppDIR, templateFolderName)
	folder, err := os.Open(templateFolder)
	if err != nil {
		log.Println("Error opening template folder", err)
		return
	}

	files, err := folder.Readdir(0)
	if err != nil {
		log.Println("Error reading from template folder", err)
		return
	}

	for _, file := range files {
		templates = append(templates, filepath.Join(AppDIR, templateFolderName, file.Name()))
	}
	return templates
}

func listProjectLogsInDir(dirName string) []projectLogListing {
	fileInfo, err := os.Stat(dirName)
	logs := []projectLogListing{}
	if err != nil {
		log.Printf("An error: %s; occurred with listing logs for %s\n", err, dirName)
		return logs
	}

	if !fileInfo.IsDir() {
		return append(logs, projectLogListing{Name: dirName, Active: ci.ActiveCISession(dirName)})
	}

	dir, err := os.Open(dirName)
	if err != nil {
		log.Printf("An error: %s; occurred will opening dir %s", err, dirName)
		return logs
	}

	files, err := dir.Readdir(0)
	if err != nil {
		log.Printf("An error: %s; occurred will reading files from dir %s", err, dirName)
		return logs
	}

	for _, file := range files {
		fileFullName := filepath.Join(dirName, file.Name())
		if file.IsDir() {
			//logs = append(logs, listProjectLogsInDir(fileFullName+"/")...)
		} else {
			name, _ := filepath.Rel(ci.LogDIR, fileFullName)
			name = strings.Replace(name, ci.LogFileExt, "", 1)
			logs = append(logs, projectLogListing{Name: name, Active: ci.ActiveCISession(fileFullName)})
		}
	}
	return logs
}

func getUserProjectsWithSubscriptionInfo(token, webhookPath string) []repoWithSubscriptionInfo {
	client := newGithubClient(token)
	repos := []repoWithSubscriptionInfo{}
	params := vcs.GithubRequestParams{CallbackURL: webhookPath}

	for _, repo := range client.UserRepos() {
		params.Owner = *(repo.Owner.Login)
		params.Repo = *(repo.Name)

		repos = append(repos, repoWithSubscriptionInfo{client.IsRepoSubscribed(params), repo})
	}

	return repos
}

func getProject(token, owner, project string) (*github.Repository, error) {
	client := newGithubClient(token)
	payload := vcs.GithubRequestParams{
		Owner: owner,
		Repo:  project,
	}

	return client.Repo(payload)
}

func fetchSession(r *http.Request) (*sessions.Session, error) {
	return SessionStore.Get(r, sessionName)
}

func addFlashMsg(msg string, w http.ResponseWriter, r *http.Request) {
	session, _ := fetchSession(r)
	session.AddFlash(msg)
	session.Save(r, w)
}

func newGithubClient(token string) *vcs.GithubClient {
	return vcs.NewGithubClient(token)
}
