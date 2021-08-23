package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/chromaui/gitlab-webhook-proxy/pkg/githost"
	"github.com/chromaui/gitlab-webhook-proxy/pkg/queue"
	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
)

const (
	DefaultQueueWaitTimeInSeconds = int64(3)

	ErrRequiredVariableString     = "environment variable required: %v"
	ErrFailedToParseIntegerString = "failed to parse variable to an integer: %v"
)

type Environment struct {
	ctx             context.Context
	gitlabUrl       string
	gitlabToken     string
	awsQueueUrl     string
	awsRegion       string
	waitTimeSeconds int64
	log             logr.Logger
}

func main() {
	env, err := getEnvironment()
	if err != nil {
		log.Fatalf("failed to setup environment: %v", err)
	}

	// setup queue and gitlab clients
	queueClient, err := queue.NewClient(env.ctx, env.log, env.awsQueueUrl, env.awsRegion, env.waitTimeSeconds)
	if err != nil {
		log.Fatalf("failed to setup queue client: %v", err)
	}

	gitlabClient, err := githost.NewClient(env.ctx, env.log, *http.DefaultClient, env.gitlabUrl, env.gitlabToken)
	if err != nil {
		log.Fatalf("failed to setup gitlab client: %v", err)
	}

	env.log.Info("application setup, reading and writing messages...")

	// infinitely read from the queue and updating gitlab
	for {
		env.log.V(1).Info("getting messages from queue")
		messages, err := queueClient.ReceiveMessages()
		if err != nil {
			log.Printf("failed to get messages: %v", err)
		}

		if len(messages) == 0 {
			env.log.V(1).Info("no messages on queue, reading again...")
			continue
		}

		env.log.V(1).Info("sending messages to gitlab")
		err = gitlabClient.SendMessages(messages, queueClient.DeleteMessage)
		if err != nil {
			log.Printf("failed to send messages: %v", err)
		}
	}
}

// getEnvironment parses out various environment variables into a single struct and returns said struct
func getEnvironment() (*Environment, error) {
	ctx := context.Background()

	awsQueueUrl := os.Getenv("AWS_QUEUE_URL")
	if awsQueueUrl == "" {
		return nil, fmt.Errorf(ErrRequiredVariableString, "AWS_QUEUE_URL")
	}

	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		return nil, fmt.Errorf(ErrRequiredVariableString, "AWS_REGION")
	}

	gitlabUrl := os.Getenv("GITLAB_URL")
	if gitlabUrl == "" {
		return nil, fmt.Errorf(ErrRequiredVariableString, "GITLAB_URL")
	}

	gitlabToken := os.Getenv("GITLAB_TOKEN")
	if gitlabToken == "" {
		return nil, fmt.Errorf(ErrRequiredVariableString, "GITLAB_TOKEN")
	}

	// set default and overwrite based on the environment variable
	waitTimeSeconds := DefaultQueueWaitTimeInSeconds
	waitTimeSecondsStr := os.Getenv("DELAY_SECONDS")
	if waitTimeSecondsStr != "" {
		seconds, err := strconv.ParseInt(waitTimeSecondsStr, 10, 0)
		if err != nil {
			return nil, fmt.Errorf(ErrFailedToParseIntegerString, "DELAY_SECOND")
		}

		waitTimeSeconds = seconds
	}

	logVerbosity := os.Getenv("LOG_VERBOSITY")
	if logVerbosity != "" {
		verbosity, err := strconv.ParseInt(logVerbosity, 10, 0)
		if err != nil {
			return nil, fmt.Errorf(ErrFailedToParseIntegerString, "LOG_VERBOSITY")
		}
		stdr.SetVerbosity(int(verbosity))
	}
	logger := stdr.New(log.Default())

	return &Environment{
		ctx:             ctx,
		gitlabUrl:       gitlabUrl,
		gitlabToken:     gitlabToken,
		awsQueueUrl:     awsQueueUrl,
		awsRegion:       awsRegion,
		waitTimeSeconds: waitTimeSeconds,
		log:             logger,
	}, nil
}
