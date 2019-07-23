// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dashboard

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type oauthHandler struct {
	clientID     string
	clientSecret string
	secretState  string
}

// Support for GitHub oauth flow
func newOAuthHandler(clientID string, clientSecret string) *oauthHandler {
	// secret state for OAuth exchanges
	secretState := make([]byte, 32)
	_, _ = rand.Read(secretState)

	ss := base64.StdEncoding.EncodeToString(secretState)

	return &oauthHandler{
		clientID:     clientID,
		secretState:  ss,
		clientSecret: clientSecret,
	}
}

func (h *oauthHandler) ServeLogin(w http.ResponseWriter, r *http.Request) error {
	v := url.Values{}
	v.Add("client_id", h.clientID)
	v.Add("scope", "user,repo")
	v.Add("state", h.secretState)

	redirectURL := "https://github.com/login/oauth/authorize?" + v.Encode()
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
	return nil
}

func (h *oauthHandler) ServeCallback(w http.ResponseWriter, r *http.Request) error {
	httpClient := http.Client{}

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("unable to parse query: %v", err)
	}

	if r.FormValue("state") != h.secretState {
		return fmt.Errorf("unable to verify request state")
	}

	v := url.Values{}
	v.Add("client_id", h.clientID)
	v.Add("client_secret", h.clientSecret)
	v.Add("code", r.FormValue("code"))

	url := "https://github.com/login/oauth/access_token?" + v.Encode()
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("unable to create request: %v", err)
	}
	// ask for the response in JSON
	req.Header.Set("accept", "application/json")

	// send out the request to GitHub for the access token
	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to contact GitHub: %v", err)
	}
	defer res.Body.Close()

	var t oauthAccessResponse
	if err := json.NewDecoder(res.Body).Decode(&t); err != nil {
		return fmt.Errorf("unable to parse response from GitHub: %v", err)
	}

	// finally, have GitHub redirect the user to the home page, passing the access token to the page
	w.Header().Set("Location", "/?access_token="+t.AccessToken)
	w.WriteHeader(http.StatusFound)

	return nil
}

type oauthAccessResponse struct {
	AccessToken string `json:"access_token"`
}
