package githost

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/chromaui/gitlab-webhook-proxy/mocks"
	"github.com/chromaui/gitlab-webhook-proxy/pkg/types"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBuildStructConversion(t *testing.T) {
	input := removeWhitespace(`{
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
	}`)

	var build types.BuildMessage
	err := json.Unmarshal([]byte(input), &build)
	assert.NoError(t, err)

	// setup a new encoder and disable HTML escapes so '&' is not encoded as '\u0026'
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)

	err = encoder.Encode(build)
	assert.NoError(t, err)

	output := removeWhitespace(buffer.String())

	assert.Equal(t, input, output)
}

func TestNewClient(t *testing.T) {
	client, err := NewClient(context.Background(), logr.Discard(), *http.DefaultClient, "https://chromatic.example.com/", "randomToken")
	assert.NoError(t, err)
	assert.Falsef(t, strings.HasSuffix(client.hostUrl, "/"), "An ending slash should be trimmed when creating a new client")
}

func TestSendMessages(t *testing.T) {
	testClient := &Client{
		ctx:     context.Background(),
		log:     logr.Discard(),
		hostUrl: "https://chromatic.example.com",
		token:   "randomToken",
	}
	testBuilds := []types.BuildMessage{
		{
			Event:         "",
			Build:         types.Build{},
			RepoId:        "",
			ReceiptHandle: "",
		},
	}

	t.Run("doesn't return an error if the response is 201 (created)", func(t *testing.T) {
		mockHttpClient := new(mocks.HttpClient)
		mockHttpClient.
			On("Do", mock.Anything).
			Return(&http.Response{StatusCode: 200}, nil)
		testClient.httpClient = mockHttpClient

		err := testClient.SendMessages(testBuilds, func(_ string) error {
			return nil
		})
		assert.NoError(t, err)
	})

	t.Run("returns an error if the response is 403 (forbidden)", func(t *testing.T) {
		mockHttpClient := new(mocks.HttpClient)
		mockHttpClient.
			On("Do", mock.Anything).
			Return(&http.Response{StatusCode: 403, Body: newReadCloser("forbidden")}, nil)
		testClient.httpClient = mockHttpClient

		err := testClient.SendMessages(testBuilds, func(_ string) error {
			return nil
		})
		assert.Error(t, err)
	})
}

func TestGetStatusUpdate(t *testing.T) {
	t.Run("build passed", func(t *testing.T) {
		update := getStatusUpdate(types.BuildMessage{
			Build: types.Build{
				Status: "PASSED",
				Number: 123,
			},
		})
		assert.Equal(t, "success", update.State)
		assert.Equal(t, "Build 123 passed unchanged.", update.Description)
	})

	t.Run("build accepted", func(t *testing.T) {
		update := getStatusUpdate(types.BuildMessage{
			Build: types.Build{
				Status: "ACCEPTED",
				Number: 123,
			},
		})
		assert.Equal(t, "success", update.State)
		assert.Equal(t, "Build 123 accepted.", update.Description)
	})

	t.Run("build pending", func(t *testing.T) {
		update := getStatusUpdate(types.BuildMessage{
			Build: types.Build{
				Status:      "PENDING",
				Number:      123,
				ChangeCount: 2,
			},
		})
		assert.Equal(t, "pending", update.State)
		assert.Equal(t, "Build 123 has 2 changes that must be accepted.", update.Description)
	})

	t.Run("build denied", func(t *testing.T) {
		update := getStatusUpdate(types.BuildMessage{
			Build: types.Build{
				Status: "DENIED",
				Number: 123,
			},
		})
		assert.Equal(t, "failed", update.State)
		assert.Equal(t, "Build 123 denied.", update.Description)
	})

	t.Run("build broken", func(t *testing.T) {
		update := getStatusUpdate(types.BuildMessage{
			Build: types.Build{
				Status: "BROKEN",
				Number: 123,
			},
		})
		assert.Equal(t, "failed", update.State)
		assert.Equal(t, "Build 123 failed to render.", update.Description)
	})

	t.Run("build failed", func(t *testing.T) {
		update := getStatusUpdate(types.BuildMessage{
			Build: types.Build{
				Status: "FAILED",
				Number: 123,
			},
		})
		assert.Equal(t, "failed", update.State)
		assert.Equal(t, "Build 123 has suffered a system error. Please try again.", update.Description)
	})
}

// removeWhitespace removes all whitespace characters so we can accurately compare string values
func removeWhitespace(input string) string {
	output := strings.ReplaceAll(input, "\t", "")
	output = strings.ReplaceAll(output, "\n", "")
	output = strings.ReplaceAll(output, " ", "")
	return output
}

// newReadCloser is a utility function to generate a ReadCloser for response bodies
func newReadCloser(body string) io.ReadCloser {
	reader := strings.NewReader(body)
	return io.NopCloser(reader)
}
