package githost

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/chromaui/gitlab-webhook-proxy/pkg/types"
	"github.com/go-logr/logr"
)

type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	ctx        context.Context
	log        logr.Logger
	httpClient HttpClient
	hostUrl    string
	token      string
}

type StatusUpdate struct {
	State       string `json:"state,omitempty"`
	Description string `json:"description,omitempty"`
}

// NewClient sets up and returns a new Client struct
func NewClient(ctx context.Context, log logr.Logger, httpClient http.Client, hostUrl, token string) (*Client, error) {
	// ensure to remove the ending slash so appending to the URL in the future is easier to read
	hostUrl = strings.TrimSuffix(hostUrl, "/")

	return &Client{
		ctx:        ctx,
		log:        log,
		httpClient: &httpClient,
		hostUrl:    hostUrl,
		token:      token,
	}, nil
}

// SendMessages sends build messages to the Client's hostUrl and calls a callback when messages are successfully
// sent. For the initial implementation, this is used to delete messages from the SQS queue so they're not triggered
// multiple times.
func (gc *Client) SendMessages(builds []types.BuildMessage, successCallback func(string) error) error {
	for _, build := range builds {
		gc.log.V(1).Info("setting commit status", "build", build)
		err := gc.setCommitStatus(build)
		if err != nil {
			return fmt.Errorf("failed to set commit status: %w", err)
		}

		err = successCallback(build.ReceiptHandle)
		if err != nil {
			return fmt.Errorf("callback failed: %w", err)
		}
		gc.log.V(1).Info("callback successful")
	}

	return nil
}

// setCommitStatus calls the git provider (GitLab in this case) to add buildMessage details to a specific commit.
//
// https://docs.gitlab.com/ee/api/commits.html#post-the-build-status-to-a-commit
func (gc *Client) setCommitStatus(message types.BuildMessage) error {
	gc.log.V(1).Info("building status update struct from build", "build", message)
	statusUpdate := getStatusUpdate(message)
	gc.log.V(1).Info("built status update", "build status", message.Build.Status, "status update", statusUpdate)

	queryString := url.Values{}
	queryString.Add("context", "UI Tests")
	queryString.Add("target_url", message.Build.WebURL)
	queryString.Add("state", statusUpdate.State)
	queryString.Add("description", statusUpdate.Description)
	gc.log.V(1).Info("built query string", "query string", queryString.Encode())

	urlWithQueryString := fmt.Sprintf(
		"%v/projects/%v/statuses/%v?%v",
		gc.hostUrl,
		message.RepoId,
		message.Build.Commit,
		queryString.Encode(),
	)

	gc.log.V(1).Info("creating request for url", "url", urlWithQueryString)
	req, err := http.NewRequestWithContext(gc.ctx, "POST", urlWithQueryString, nil)
	if err != nil {
		return fmt.Errorf("failed to setup new request: %w", err)
	}

	token := fmt.Sprintf("Bearer %v", gc.token)
	req.Header.Add("Authorization", token)

	gc.log.V(1).Info("sending post request", "url", req.URL.String())
	res, err := gc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to completed request: %w", err)
	}
	gc.log.V(1).Info("response from gitlab", "status code", res.StatusCode, "status", res.Status)

	if res.StatusCode >= 400 {
		defer res.Body.Close()
		errBody, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed to parse body: %w", err)
		}

		if !filterErrorResponse(res, string(errBody)) {
			return fmt.Errorf("failed to post to gitlab: %v, %#v", string(errBody), res)
		}
	}

	return nil
}

// getStatusUpdate builds and returns a StatusUpdate struct based on the details from a BuildMessage
func getStatusUpdate(message types.BuildMessage) StatusUpdate {
	if message.Build.Status == "FAILED" {
		return StatusUpdate{
			State:       "failed",
			Description: fmt.Sprintf("Build %v has suffered a system error. Please try again.", message.Build.Number),
		}
	}

	if message.Build.Status == "BROKEN" {
		return StatusUpdate{
			State:       "failed",
			Description: fmt.Sprintf("Build %v failed to render.", message.Build.Number),
		}
	}

	if message.Build.Status == "DENIED" {
		return StatusUpdate{
			State:       "failed",
			Description: fmt.Sprintf("Build %v denied.", message.Build.Number),
		}
	}

	if message.Build.Status == "PENDING" {
		return StatusUpdate{
			State:       "pending",
			Description: fmt.Sprintf("Build %v has %v changes that must be accepted.", message.Build.Number, message.Build.ChangeCount),
		}
	}

	if message.Build.Status == "ACCEPTED" {
		return StatusUpdate{
			State:       "success",
			Description: fmt.Sprintf("Build %v accepted.", message.Build.Number),
		}
	}

	if message.Build.Status == "PASSED" {
		return StatusUpdate{
			State:       "success",
			Description: fmt.Sprintf("Build %v passed unchanged.", message.Build.Number),
		}
	}

	return StatusUpdate{}
}

// filterErrorResponse returns true if the response we get from GitLab isn't one that we need to handle
func filterErrorResponse(response *http.Response, responseBody string) bool {
	// GitLab has a fun issue that throws the following error with "HTTP 400 Bad Request" but it's not a true error as
	// everything functions within GitLab just fine. We can ignore these for now.
	if response.StatusCode == 400 && strings.HasPrefix(responseBody, "Cannot transition status via :enqueue from :pending") {
		return true
	}

	return false
}
