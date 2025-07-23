package mail

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/freekieb7/gravel/auth/oauth"
)

type Mailer interface {
	Send(mail ...Mail) error
}

type Payload struct {
	Message         Message `json:"message"`
	SaveToSentItems bool    `json:"saveToSentItems"`
}

type Message struct {
	Subject      string        `json:"subject"`
	Body         Body          `json:"body"`
	ToRecipients []ToRecipient `json:"toRecipients"`
}

type Body struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

type ToRecipient struct {
	EmailAddress EmailAddress `json:"emailAddress"`
}

type EmailAddress struct {
	Address string `json:"address"`
}

func NewMicrosoftMailer(tenantId, clientId, clientSecret, userId string) Mailer {
	return &microsoftMailer{
		tenantId:     tenantId,
		clientId:     clientId,
		clientSecret: clientSecret,
		userId:       userId,
	}
}

type microsoftMailer struct {
	tenantId, clientId, clientSecret, userId string
}

func (mailer *microsoftMailer) Send(mails ...Mail) error {
	client := oauth.MicrosoftClient{
		ClientId:     mailer.clientId,
		ClientSecret: mailer.clientSecret,
		TokenUrl:     fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", mailer.tenantId),
	}

	token, err := client.Token()
	if err != nil {
		return errors.Join(errors.New("getting token failed"), err)
	}

	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/sendMail", mailer.userId)
	payload, err := json.Marshal(Payload{
		Message: Message{
			Subject: "Test",
			Body: Body{
				ContentType: "Text",
				Content:     "Hi, this is a test",
			},
			ToRecipients: []ToRecipient{
				{
					EmailAddress: EmailAddress{
						Address: "freekieb7@hotmail.com",
					},
				},
			},
		},
		SaveToSentItems: false,
	})
	if err != nil {
		return errors.Join(errors.New("payload marshal failed"), err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return errors.Join(errors.New("buffer fucked up"), err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	mailclient := &http.Client{}
	resp, err := mailclient.Do(req)
	if err != nil {
		return errors.Join(errors.New("sending mail failed"), err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("resp.Body.Close error", "error", err)
		}
	}()

	if _, err = io.ReadAll(resp.Body); err != nil {
		return errors.Join(errors.New("reading response body failed"), err)
	}

	if resp.StatusCode != 202 {
		return errors.Join(errors.New("mail rejected"), err)
	}

	return nil
}
