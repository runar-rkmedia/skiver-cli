package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/runar-rkmedia/go-common/logger"
	"github.com/runar-rkmedia/skiver/handlers"
	"github.com/runar-rkmedia/skiver/models"
	"github.com/runar-rkmedia/skiver/types"
)

type Api struct {
	l        logger.AppLogger
	endpoint string
	cookies  []*http.Cookie
	login    *types.LoginResponse
	client   *http.Client
}

func NewAPI(l logger.AppLogger, endpoint string) Api {
	c := http.Client{Timeout: time.Minute}
	api := Api{
		l:        l,
		endpoint: strings.TrimSuffix(endpoint, "/"),
		client:   &c,
	}
	if l.HasDebug() {
		l.Debug().Str("uri", endpoint).Msg("Using skiver-api")
	}
	return api
}

func (a *Api) SetToken(token string) {
	c := http.Cookie{
		Name:  "token",
		Value: token,
	}
	a.cookies = append(a.cookies, &c)
}
func (a *Api) Login(username, password string) error {
	if username == "" || password == "" {
		return fmt.Errorf("Missing username/password")
	}
	payload := struct{ Username, Password string }{Username: username, Password: password}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("Failed to marshal login-payload: %w", err)
	}
	r, err := http.NewRequest(http.MethodPost, a.endpoint+"/api/login/", bytes.NewBuffer(b))
	if err != nil {
		return fmt.Errorf("failed to create login-request: %w", err)
	}
	r.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(r)
	if err != nil {
		return fmt.Errorf("login-request failed: %w", err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed reading body of login-request: %w", err)
	}
	if res.StatusCode >= 300 {
		var j models.APIError
		err = json.Unmarshal(body, &j)
		if err == nil {
			return fmt.Errorf("Login-request return a %d-response: %s (%s) %#v", res.StatusCode, j.Error.Message, j.Error.Code, j.Details)
		}

		return fmt.Errorf("Login-request return a %d-response: %s", res.StatusCode, string(body))
	}
	var j types.LoginResponse
	err = json.Unmarshal(body, &j)
	if err != nil {
		return fmt.Errorf("failed reading body of login-request: %w", err)
	}
	a.cookies = res.Cookies()
	if a.l.HasDebug() {
		a.l.Debug().
			Int("statusCode", res.StatusCode).
			Str("path", res.Request.URL.String()).
			Str("method", res.Request.Method).
			Interface("login-response", j).
			Msg("Result of request")
	}
	return nil

}
func (a Api) Import(projectName string, kind string, locale string, reader io.Reader) error {
	if len(a.cookies) == 0 {
		return fmt.Errorf("Not logged in")
	}
	r, err := http.NewRequest(http.MethodPost, a.endpoint+"/api/import/"+kind+"/"+projectName+"/"+locale, reader)
	if err != nil {
		return fmt.Errorf("failed to create import-request: %w", err)
	}
	r.Header.Add("Content-Type", "application/json")
	for _, c := range a.cookies {
		r.AddCookie(c)
	}

	res, err := http.DefaultClient.Do(r)
	if err != nil {
		return fmt.Errorf("import-request failed: %w", err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed reading body of import-request: %w", err)
	}
	if res.StatusCode >= 300 {
		var j models.APIError
		err = json.Unmarshal(body, &j)
		if err == nil {
			return fmt.Errorf("import-request returned %d-response: %s (%s) %#v", res.StatusCode, j.Error.Message, j.Error.Code, j.Details)
		}

		return fmt.Errorf("import-request returned %d-response: %s", res.StatusCode, string(body))
	}
	var j handlers.ImportResult
	err = json.Unmarshal(body, &j)
	if err != nil {
		return fmt.Errorf("failed reading body of import-request: %w", err)
	}
	if a.l.HasDebug() {
		a.l.Debug().
			Int("statusCode", res.StatusCode).
			Str("path", res.Request.URL.String()).
			Str("method", res.Request.Method).
			Interface("import-warnings", j.Warnings).
			Int("translation-creations", len(j.Changes.TranslationCreations)).
			Int("category-creations", len(j.Changes.CategoryCreations)).
			Int("translation-value-creations", len(j.Changes.TranslationValueUpdates)).
			Int("translation-value-creations", len(j.Changes.TranslationsValueCreations)).
			Msg("Result of request")
	}
	return nil

}

func (a Api) Export(projectName string, format string, locale string, writer io.Writer) error {
	if len(a.cookies) == 0 {
		return fmt.Errorf("Not logged in")
	}
	r, err := http.NewRequest(http.MethodGet, a.endpoint+"/api/export/", nil)
	if err != nil {
		return fmt.Errorf("failed to create export-request: %w", err)
	}
	q := r.URL.Query()
	q.Set("format", format)
	q.Set("locale", locale)
	q.Set("project", projectName)
	r.URL.RawQuery = q.Encode()
	r.Header.Add("Content-Type", "application/json")
	for _, c := range a.cookies {
		r.AddCookie(c)
	}

	res, err := http.DefaultClient.Do(r)
	if err != nil {
		return fmt.Errorf("export-request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed reading body of export-request: %w", err)
		}
		var j models.APIError
		err = json.Unmarshal(body, &j)
		if err == nil {
			return fmt.Errorf("export-request return a %d-response: %s (%s) %#v", res.StatusCode, j.Error.Message, j.Error.Code, j.Details)
		}

		return fmt.Errorf("export-request return a %d-response: %s", res.StatusCode, string(body))
	}
	written, err := io.Copy(writer, res.Body)
	if a.l.HasDebug() {
		a.l.Debug().
			Int("statusCode", res.StatusCode).
			Str("path", res.Request.URL.String()).
			Str("method", res.Request.Method).
			Int64("written-bytes", written).
			Str("written-text", humanize.Bytes(uint64(written))).
			Msg("Result of request")
	}
	return nil

}
