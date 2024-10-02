package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
)

const (
	StatusOK    = "ok"
	StatusError = "error"
)

const (
	MsgGitPullDone = "Git pull done"
)

type GitHubWebhookPayload struct {
	Action string `json:"action"`
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
}

type WebhookResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

var (
	allowedBranches []string
	repoPathMap     map[string]string
)

func init() {
	allowedBranches = []string{
		"main",
		"gh-pages",
		"cf-pages",
		"al-pages",
	}

	repoPathMap = map[string]string{
		"repo1": "/path/to/repo1",
		"repo2": "/path/to/repo2",
	}
}

func isAllowedBranch(branch string) bool {
	for _, allowedBranch := range allowedBranches {
		if branch == allowedBranch {
			return true
		}
	}
	return false
}

func pullBranch(branch, path string) error {
	// Switch to the target branch
	checkoutCmd := exec.Command("git", "checkout", branch)
	checkoutCmd.Dir = path
	checkoutOutput, err := checkoutCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to checkout branch %s (path: %s): %s\nOutput: %s", branch, path, err, checkoutOutput)
	}
	log.Printf("Successfully checked out branch %s (path: %s)\nCheckout Output:\n%s", branch, path, checkoutOutput)

	// Execute git pull
	pullCmd := exec.Command("git", "pull", "origin", branch)
	pullCmd.Dir = path
	pullOutput, err := pullCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull branch %s (path: %s): %s\nOutput: %s", branch, path, err, pullOutput)
	}
	log.Printf("Successfully pulled branch %s (path: %s)\nPull Output:\n%s", branch, path, pullOutput)

	return nil
}

func handleGitPull(payload GitHubWebhookPayload) error {
	branch := payload.Branch
	repo := payload.Repo

	if branch == "" || repo == "" {
		return fmt.Errorf("missing required parameters")
	}

	if !isAllowedBranch(branch) {
		return fmt.Errorf("branch is not in the allowed list: %s", branch)
	}

	absPath, ok := repoPathMap[repo]
	if !ok {
		return fmt.Errorf("unknown repository: %s", repo)
	}

	// Check if the path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("specified path does not exist: %s", absPath)
	}

	return pullBranch(branch, absPath)
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errMsg := "Only POST requests are accepted"
		log.Println(errMsg)
		http.Error(w, createErrorJSON(errMsg), http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		errMsg := "Failed to read request body: " + err.Error()
		log.Println(errMsg)
		http.Error(w, createErrorJSON(errMsg), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload GitHubWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		errMsg := "Failed to parse JSON: " + err.Error()
		log.Println(errMsg)
		http.Error(w, createErrorJSON(errMsg), http.StatusBadRequest)
		return
	}

	response := WebhookResponse{}
	switch payload.Action {
	case "git-pull":
		if err := handleGitPull(payload); err != nil {
			errMsg := "Failed to handle git-pull: " + err.Error()
			log.Println(errMsg)
			http.Error(w, createErrorJSON(errMsg), http.StatusInternalServerError)
			return
		}
		response.Status = StatusOK
		response.Message = MsgGitPullDone
	default:
		errMsg := "Unsupported action: " + payload.Action
		log.Println(errMsg)
		http.Error(w, createErrorJSON(errMsg), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		errMsg := "Failed to encode response: " + err.Error()
		log.Println(errMsg)
		http.Error(w, createErrorJSON(errMsg), http.StatusInternalServerError)
		return
	}
	log.Printf("Successfully processed request: %s", response.Message)
}

func createErrorJSON(message string) string {
	errorResponse := WebhookResponse{
		Status:  StatusError,
		Message: message,
	}
	jsonResponse, err := json.Marshal(errorResponse)
	if err != nil {
		log.Printf("Failed to marshal error response: %v", err)
		return `{"status":"error","message":"Internal server error"}`
	}
	return string(jsonResponse)
}

func main() {
	var port string
	flag.StringVar(&port, "port", "", "Specify the server listening port")
	flag.Parse()

	if port == "" {
		port = os.Getenv("PORT")
	}

	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/webhook", webhookHandler)
	log.Printf("Starting to listen on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
