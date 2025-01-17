package ci

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// LogFileExt is the file extension used for log files
	LogFileExt = "e14a5940cc6e873f53d82ce346e7ed6b8ecbdd1d.log"
	BisectName = "bisect.txt"
	BackupName = "backup"
)

var (
	// ciDIR is the absolute path to the CI directory
	ciDIR string
	// LogDIR is the absolute path to the CI log directory
	LogDIR string
	// List of supported languages
	// and the available docker image version
	availableImages = map[string]string{
		"ruby":       "xovox/sicuro_ruby:0.2",
		"javascript": "xovox/sicuro_javascript:0.2",
		"go":         "ci_image:1.16",
	}
)

func init() {
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	ciDIR = filepath.Join(os.Getenv("ROOT_DIR"), "ci")
	LogDIR = filepath.Join(ciDIR, "logs")

}

// JobDetails contains necessary information required to run tests for a given project
type JobDetails struct {
	// LogFileName is the name to be used for the test output log.
	// It should always be the commit hash e.g 5eace776ec66a70b2775f4bbb9e2b2847331b0a9
	// or branch name e.g master
	LogDirPath  string
	LogFileName string
	logFilePath string
	// IsRevert show that jov is running for commit revert
	IsRevert string
	// ProjectRespositoryName is the name of the project's repository on the VCS
	// It's used when cloning the project in test container
	ProjectRespositoryName string
	// ProjectBranch is the target branch to run the tests on
	// It could also be a commit hash if the target is a particular commit
	ProjectBranch string
	// ProjectRespositoryURL is the SSH url for pull the code from the VCS
	ProjectRepositoryURL string
	// ProjectLanguage is the programming language the project is written in
	// This would be used to determine the docker image for running the tests
	ProjectLanguage string
	// UpdateBuildStatus is a callback function that would be executed with updates of the test
	// It would be executed with the build status pending, failure, success as argument
	// Once the tests starts, it's executed with the pending status argument
	// At test completion it would be executed again with the result status: success or failure
	UpdateBuildStatus func(string)
}

func init() {
	// set ci path env variable
	os.Setenv("CI_DIR", ciDIR)
}

// Run triggers the CI server for the given job
// It builds the absolute path to the job log file, creating necessary parent directories
// It terminates if a routine is currently active for the given job
// Otherwise, sets up a new routine for the job
func Run(job *JobDetails) {
	job.logFilePath = filepath.Join(LogDIR, fmt.Sprintf("%s%s", job.LogFileName, LogFileExt))
	err := createDirFor(job.logFilePath)
	if err != nil {
		log.Println("Couldn't create directory for job: ", err)
		return
	}

	// ensure file is not being written to
	if ActiveCISession(job.logFilePath) {
		log.Println("A job is currently in progress: ", job.logFilePath)
		return
	}

	// prepare log file i.e clear file content or create new file
	if err := exec.Command("bash", "-c", "> "+job.logFilePath).Run(); err != nil {
		log.Printf("Error: %s occurred while trying to clear logfile %s\n", err, job.logFilePath)
		return
	}

	job.ProjectLanguage = strings.ToLower(job.ProjectLanguage)
	if !supportedLanguage(job.ProjectLanguage) {
		log.Println("Project Language is currently not supported")
		return
	}

	log.Printf("Running job: %v\n", job)
	go runCI(job)
}

func createDirFor(fileName string) error {
	dir, file := filepath.Split(fileName)
	log.Printf("Making dir: %s for file: %s\n", dir, file)
	return os.MkdirAll(dir, 0755)
}

func runCI(job *JobDetails) {
	logFile, err := os.OpenFile(job.logFilePath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		log.Printf("Error %s occurred while opening log file: %s\n", err, job.logFilePath)
		job.updateBuildStatus("error")
		return
	}
	err = os.MkdirAll(filepath.Join(LogDIR, job.LogDirPath, BackupName), 0755)
	if err != nil {
		log.Println("Error while creating log directory", err)
	}
	bisectFile, err := os.OpenFile(filepath.Join(LogDIR, job.LogDirPath, BackupName, BisectName), os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		log.Printf("Error %s occurred while opening bisect file\n", err)
		job.updateBuildStatus("error")
		return
	}
	bisectCont := readByte(bisectFile)
	defer logFile.Close()
	defer bisectFile.Close()
	job.updateBuildStatus("pending")

	containerImg := availableImages[job.ProjectLanguage]
	isRevert, err := strconv.ParseBool(job.IsRevert)
	os.Setenv("PROJECT_DIR", filepath.Join(LogDIR, job.LogDirPath))
	if err != nil {
		log.Printf("Job will started for build")
	}
	if isRevert {
		containerImg = strings.Join([]string{"backup_", containerImg}, "")
		cmmt := parse(bisectCont)
		if cmmt != "" {
			os.Setenv("GOOD_COMMIT", cmmt)
		}
	}
	cmd := exec.Command("bash", "-c", fmt.Sprintf("%s '%s' %s", filepath.Join(ciDIR, "run.sh"), prepareEnvVars(job), containerImg))
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	err = cmd.Run()

	msg := "Build completed successfully"
	status := "success"
	log.Println("Exit code: ", err)
	if err != nil {
		msg = fmt.Sprintf("Build failed with exit code: %s. You may revert changes <p><a href='/run?repo=%s&revert=1'>Revert commit</a><p>", err, job.LogFileName)
		status = "failure"
	}

	job.updateBuildStatus(status)
	logFile.WriteString(fmt.Sprintf("<h4>%s</h4>", msg))
	logFile.WriteString(fmt.Sprintf("<p><a href='/run?repo=%s'>Rebuild</a><p>", job.LogFileName))
	findCommit(bisectFile, bisectCont, job.ProjectBranch, status)
	logFile.Close()
	bisectFile.Close()
}

func (job *JobDetails) updateBuildStatus(status string) {
	if job.UpdateBuildStatus != nil {
		job.UpdateBuildStatus(status)
	}
}

func supportedLanguage(lang string) (ok bool) {
	_, ok = availableImages[lang]
	return
}

// ActiveCISession returns true if a ci session is active
// it returns false otherwise
func ActiveCISession(logFile string) bool {
	cmd := exec.Command("lsof", logFile)
	return cmd.Run() == nil
}

func prepareEnvVars(job *JobDetails) (vars string) {
	vars = fmt.Sprintf("%s -e %s=%s", vars, "PROJECT_BRANCH", job.ProjectBranch)
	vars = fmt.Sprintf("%s -e %s=%s", vars, "COMMIT", job.ProjectBranch[:7])
	vars = fmt.Sprintf("%s -e %s=%s", vars, "PROJECT_REPOSITORY_URL", job.ProjectRepositoryURL)
	vars = fmt.Sprintf("%s -e %s=%s", vars, "PROJECT_REPOSITORY_NAME", job.ProjectRespositoryName)
	vars = fmt.Sprintf("%s -e %s=%s", vars, "PROJECT_LANGUAGE", job.ProjectLanguage)
	vars = fmt.Sprintf("%s -e %s=%s", vars, "GITHUB_TOKEN", os.Getenv("GITHUB_TOKEN"))
	vars = fmt.Sprintf("%s -e %s=%s", vars, "EMAIL", os.Getenv("EMAIL"))
	vars = fmt.Sprintf("%s -e %s=%s", vars, "USER_NAME", os.Getenv("USER_NAME"))
	vars = fmt.Sprintf("%s -e %s=%s", vars, "GOOD_COMMIT", os.Getenv("GOOD_COMMIT"))
	return
}
