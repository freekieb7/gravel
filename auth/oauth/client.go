package oauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type MicrosoftClient struct {
	ClientId     string
	ClientSecret string
	TokenUrl     string
}

func (client *MicrosoftClient) Token() (string, error) {
	var token string

	response, err := http.PostForm(client.TokenUrl, url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {client.ClientId},
		"client_secret": {client.ClientSecret},
		"scope":         {"https://graph.microsoft.com/.default"},
	})
	if err != nil {
		return token, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return token, fmt.Errorf("bad status code %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return token, err
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return token, err
	}

	token, ok := data["access_token"].(string)
	if !ok {
		return token, errors.New("unable to get access token from response")
	}

	return token, nil
}
