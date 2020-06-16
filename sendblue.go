package sendblue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/ttacon/libphonenumber"
)

// A Client is used to send API requests to Sendblue.
type Client struct {
	APIKey    string
	SecretKey string
	Client    *http.Client
}

// NewCustomClient creates a new client from a key pair and a custom client.
func NewCustomClient(client *http.Client, api, secret string) *Client {
	return &Client{
		APIKey:    api,
		SecretKey: secret,
		Client:    client,
	}
}

// NewDefaultClient creates a new client from a key pair, using the
// default http.Client implementation.
func NewDefaultClient(api, secret string) *Client {
	return NewCustomClient(&http.Client{}, api, secret)
}

var (
	// ErrParse is returned when we are unable to parse a phone number
	// into the proper format. In this case, it's advised to show an
	// error message to the user and ask them to check their input.
	ErrParse = fmt.Errorf("sendblue: failed to parse phone number")
)

// A Message is sent or received from Sendblue.
type Message struct {
	Number  string `json:"number"`
	Content string `json:"content"`
}

// A MessageResponse is returned from Sendblue after a message request
// was sent. This is used to check for any errors.
type MessageResponse struct {
	Status        string `json:"status"`
	ErrorCode     string `json:"error_code"`
	FromNumber    string `json:"from_number"`
	MessageHandle string `json:"message_handle"`
}

// The current endpoint - subject to change.
const endpoint = "https://bluetexts-272923.uc.r.appspot.com/api/send-message"

// SendMessage sends a message to a phone number `to` with a given `body`.
// Returns the phone number that the message was sent from and an error, if any.
func (c *Client) SendMessage(to, body string) (string, error) {
	num, err := libphonenumber.Parse(to, "US")
	if err != nil {
		return "", ErrParse
	}
	buf, err := json.Marshal(Message{
		Number:  libphonenumber.Format(num, libphonenumber.E164),
		Content: body,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(buf))
	if err != nil {
		return "", fmt.Errorf("failed to create post request: %w", err)
	}
	req.Header.Set("sb-api-key-id", c.APIKey)
	req.Header.Set("sb-api-secret-key", c.SecretKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()
	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	var mresp MessageResponse
	if err := json.Unmarshal(rbody, &mresp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	if mresp.Status == "ERROR" {
		return "", fmt.Errorf("failed to send, returned error")
	}
	return mresp.FromNumber, nil
}

// ReadWebhook is a method used to process an incoming webhook from Sendblue.
func ReadWebhook(r io.ReadCloser) (*Message, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Close()
	msg := new(Message)
	if err := json.Unmarshal(data, msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal into message: %w", err)
	}
	return msg, nil
}
