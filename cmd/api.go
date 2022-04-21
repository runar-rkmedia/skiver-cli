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
	"github.com/runar-rkmedia/go-common/utils"
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
	Headers  http.Header
}

func NewAPI(l logger.AppLogger, endpoint string) Api {
	c := http.Client{Timeout: time.Minute}
	api := Api{
		l:        l,
		endpoint: strings.TrimSuffix(endpoint, "/"),
		client:   &c,
		Headers:  http.Header{},
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
	r, err := a.NewRequest(http.MethodPost, "/api/login/", bytes.NewBuffer(b))
	if err != nil {
		return fmt.Errorf("failed to create login-request: %w", err)
	}

	res, err := http.DefaultClient.Do(r)
	if err != nil {
		return fmt.Errorf("login-request failed: %w", err)
	}
	var j types.LoginResponse
	res, err = a.Do(r, &j)
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
	r, err := a.NewRequest(http.MethodPost, "/api/import/"+kind+"/"+projectName+"/"+locale, reader)
	if err != nil {
		return fmt.Errorf("failed to create import-request: %w", err)
	}

	var j handlers.ImportResult
	res, err := a.Do(r, &j)
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
	r, err := a.NewRequest(http.MethodGet, "/api/export/", nil)
	if err != nil {
		return fmt.Errorf("failed to create export-request: %w", err)
	}
	q := r.URL.Query()
	q.Set("format", format)
	q.Set("locale", locale)
	q.Set("project", projectName)
	r.URL.RawQuery = q.Encode()

	res, err := a.Do(r, nil)
	if err != nil {
		return fmt.Errorf("export-request failed: %w", err)
	}
	defer res.Body.Close()

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

// NewRequest is a thin wrapper around http.NewRequest
func (a *Api) NewRequest(method string, subpath string, body io.Reader) (*http.Request, error) {
	uri := a.endpoint + subpath
	if a.l.HasDebug() {
		a.l.Debug().
			Str("method", subpath).
			Str("subpath", subpath).
			Str("uri", uri).
			Msg("Creating request")
	}
	r, err := http.NewRequest(method, uri, body)
	if err != nil {
		return r, err
	}
	r.Header = a.Headers.Clone()
	r.Header.Set("Content-Type", "application/json")
	for _, c := range a.cookies {
		r.AddCookie(c)
	}
	return r, err
}

// Do a http.request. If j is not nil, it will unmarshal to that destination
func (a Api) Do(r *http.Request, j interface{}) (*http.Response, error) {
	reqId := r.Header.Get("X-Request-ID")
	if reqId == "" {
		id, err := utils.ForceCreateUniqueId()
		if err != nil {
			a.l.Warn().Err(err).Str("unique-id", id).Msg("an error occured when attempting to create a unique id for the request.")
		}
		reqId = id
		r.Header.Set("X-Request-ID", id)
	}
	if a.l.HasDebug() {
		a.l.Debug().
			Str("method", r.Method).
			Str("request-id", reqId).
			Interface("headers", r.Header).
			Str("uri", r.URL.String()).
			Msg("Doing request")
	}
	res, err := a.client.Do(r)
	if err != nil {
		return res, err
	}
	if a.l.HasDebug() {
		a.l.Debug().
			Str("method", r.Method).
			Str("request-id", reqId).
			Str("uri", r.URL.String()).
			Interface("headers", res.Header).
			Interface("status-code", res.StatusCode).
			Msg("Result of request")
	}
	if res.StatusCode >= 300 {
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return res, fmt.Errorf("failed reading body of request: %w", err)
		}
		var j models.APIError
		err = json.Unmarshal(body, &j)
		if err == nil {
			return res, fmt.Errorf("request returned %d-response: %s (%s) %#v", res.StatusCode, j.Error.Message, j.Error.Code, j.Details)
		}
		return res, fmt.Errorf("request returned %d-response: %s", res.StatusCode, string(body))
	}
	if j != nil {
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return res, fmt.Errorf("failed reading body of request: %w", err)
		}
		err = json.Unmarshal(body, &j)
		if err != nil {
			return res, err
		}
	}
	return res, err
}
