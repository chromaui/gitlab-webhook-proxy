package queue

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/chromaui/gitlab-webhook-proxy/mocks"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestReceiveMessages(t *testing.T) {
	messageBody := `{
		"event": "build-status-changed",
		"build": {
			"result": "SUCCESS",
			"status": "PENDING",
			"webUrl": "https://www.chromatic.com/build?appId=726FFA765F3D48EBCEFD647E0098E272&number=1",
			"commit": "41869D89223C8D9AB2A22B48686E3508C6CC0B54",
			"committerName": "",
			"branch": "main",
			"number": 1,
			"storybookUrl": "https://726FFA765F3D48EBCEFD647E0098E272-abcdefghij.chromatic.com",
			"changeCount": 5,
			"componentCount": 1,
			"specCount": 2
		},
		"repoId": "12345"
	}`
	messageReceiptHandler := "handle123"
	testMessage := &sqs.Message{
		Body:          &messageBody,
		ReceiptHandle: &messageReceiptHandler,
	}
	mockClient := new(mocks.QueueClient)
	mockClient.
		On("ReceiveMessageWithContext", mock.Anything, mock.Anything).
		Return(
			&sqs.ReceiveMessageOutput{
				Messages: []*sqs.Message{testMessage},
			},
			nil,
		)
	client := &Client{
		ctx:             context.Background(),
		log:             logr.Discard(),
		queueUrl:        "https://chromatic.example.com",
		waitTimeSeconds: int64(3),
		QueueClient:     mockClient,
	}

	messages, err := client.ReceiveMessages()
	assert.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, messageReceiptHandler, messages[0].ReceiptHandle)
	assert.Equal(t, "12345", messages[0].RepoId)
	assert.Equal(t, "SUCCESS", messages[0].Build.Result)
}

func TestDeleteMessage(t *testing.T) {
	mockClient := new(mocks.QueueClient)
	mockClient.
		On("DeleteMessage", mock.Anything).
		Return(
			nil,
			nil,
		)
	client := &Client{
		ctx:             context.Background(),
		log:             logr.Discard(),
		queueUrl:        "https://chromatic.example.com",
		waitTimeSeconds: int64(3),
		QueueClient:     mockClient,
	}

	err := client.DeleteMessage("randomHandle")
	assert.NoError(t, err)
}
