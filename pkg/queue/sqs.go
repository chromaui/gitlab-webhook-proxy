package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/chromaui/gitlab-webhook-proxy/pkg/types"
	"github.com/go-logr/logr"
)

type QueueClient interface {
	ReceiveMessageWithContext(context.Context, *sqs.ReceiveMessageInput, ...request.Option) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(*sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error)
}

type Client struct {
	ctx             context.Context
	log             logr.Logger
	queueUrl        string
	waitTimeSeconds int64

	QueueClient QueueClient
}

// NewClient sets up and returns a new Client struct
func NewClient(ctx context.Context, log logr.Logger, queueUrl string, region string, waitTimeSeconds int64) (*Client, error) {
	log.V(1).Info("setting up session with aws")
	config := aws.NewConfig().
		WithCredentials(credentials.NewEnvCredentials()).
		WithRegion(region)
	awsSession, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("failed to setup aws session: %w", err)
	}
	log.V(1).Info("setup session with aws")

	return &Client{
		ctx:             ctx,
		log:             log,
		queueUrl:        queueUrl,
		waitTimeSeconds: waitTimeSeconds,
		QueueClient:     sqs.New(awsSession),
	}, nil
}

// ReceiveMessages gathers messages from a queue (SQS in this case) and returns the parsed BuildMessage.
func (c *Client) ReceiveMessages() ([]types.BuildMessage, error) {
	c.log.V(1).Info("gathering messages from aws sqs")
	items, err := c.QueueClient.ReceiveMessageWithContext(c.ctx, &sqs.ReceiveMessageInput{
		QueueUrl:        &c.queueUrl,
		WaitTimeSeconds: &c.waitTimeSeconds,
	})
	if err != nil {
		return []types.BuildMessage{}, fmt.Errorf("failed to get messages: %w", err)
	}
	c.log.V(1).Info("gathered messages from aws sqs")

	messages := []types.BuildMessage{}
	for _, item := range items.Messages {
		c.log.V(1).Info("parsing message", "message", *item.Body)
		var message types.BuildMessage
		if err := json.Unmarshal([]byte(*item.Body), &message); err != nil {
			c.log.Error(err, "failed to parse message body", "message", *item.Body)
		}
		message.ReceiptHandle = *item.ReceiptHandle

		c.log.V(1).Info("storing sqs message in memory", "message", message)
		messages = append(messages, message)
	}

	c.log.V(1).Info("gathered all messages", "messages", messages)
	return messages, nil
}

// DeleteMessage removes a messages from a queue (SQS in this case). SQS will requeue messages every few seconds
// assuming there was an issue with the original message. Once we handle a message on the queue, we want to remove
// it to avoid a requeue.
func (c *Client) DeleteMessage(receiptHandle string) error {
	c.log.V(1).Info("Removing message from queue", "receipt handle", receiptHandle)
	_, err := c.QueueClient.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      &c.queueUrl,
		ReceiptHandle: &receiptHandle,
	})

	return err
}
