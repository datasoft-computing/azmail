package azmail

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

type mailMessage struct {
	Attachments                    []MailAttachment `json:"attachments,omitempty"`
	Content                        MailContent      `json:"content"`
	Recipients                     MailRecipients   `json:"recipients"`
	ReplyTo                        []MailAddress    `json:"replyTo,omitempty"`
	SenderAddr                     string           `json:"senderAddress"`
	UserEngagementTrackingDisabled bool             `json:"userEngagementTrackingDisabled"`
}

func (c *Client) newMailMessage(mail Mail) mailMessage {
	return mailMessage{
		mail.Attachments,
		mail.Content,
		mail.Recipients,
		nil,
		c.senderAddr,
		true,
	}
}

// SendMail sends single mail and returns the message id
func (c *Client) SendMail(mail *Mail) (string, error) {
	msg := c.newMailMessage(*mail)
	return c.sendMessage(msg)
}

// SendMails sends multiple mails. If any errors are encountered, the error is saved and later returned.
// Encountering errors does not stop later emails from being sent.
func (c *Client) SendMails(mails ...*Mail) error {
	var errs []error

	for _, mail := range mails {
		msg := c.newMailMessage(*mail)
		if _, err := c.sendMessage(msg); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

type mailResponse struct {
	ID    string              `json:"id"`
	Error errorDetailResponse `json:"error"`
}

type errorResponse struct {
	Error errorDetailResponse `json:"error"`
}

type errorDetailResponse struct {
	AdditionalInfo []struct {
		Info any    `json:"info"`
		Type string `json:"type"`
	} `json:"additionalInfo"`
	Code    string          `json:"code"`
	Details []errorResponse `json:"details"`
	Message string          `json:"message"`
	Target  string          `json:"target"`
}

func (c *Client) sendMessage(msg mailMessage) (string, error) {
	req, err := c.generateSignedMessageRequest(msg)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var (
		b         bytes.Buffer
		mailResp  mailResponse
		errorResp errorResponse
	)

	if resp.StatusCode == http.StatusAccepted {
		if _, err = b.ReadFrom(resp.Body); err != nil {
			return "", err
		}

		if err = json.Unmarshal(b.Bytes(), &mailResp); err != nil {
			return "", err
		}

		return mailResp.ID, nil
	}

	if _, err = b.ReadFrom(resp.Body); err != nil {
		return "", err
	}

	if err = json.Unmarshal(b.Bytes(), &errorResp); err != nil {
		return "", err
	}

	return "", errors.New(errorResp.Error.Message)
}
