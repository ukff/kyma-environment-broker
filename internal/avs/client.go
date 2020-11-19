package avs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type Client struct {
	httpClient *http.Client
	avsConfig  Config
	log        logrus.FieldLogger
	ctx        context.Context
}

func NewClient(ctx context.Context, avsConfig Config, log logrus.FieldLogger) (*Client, error) {
	return &Client{
		avsConfig: avsConfig,
		log:       log,

		ctx: ctx,
	}, nil
}

func (c *Client) CreateEvaluation(evaluationRequest *BasicEvaluationCreateRequest) (_ *BasicEvaluationCreateResponse, err error) {
	var responseObject BasicEvaluationCreateResponse

	objAsBytes, err := json.Marshal(evaluationRequest)
	if err != nil {
		return &responseObject, errors.Wrap(err, "while marshaling evaluation request")
	}

	request, err := http.NewRequest(http.MethodPost, c.avsConfig.ApiEndpoint, bytes.NewReader(objAsBytes))
	if err != nil {
		return &responseObject, errors.Wrap(err, "while creating request")
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.execute(request, false, true)
	if err != nil {
		return &responseObject, errors.Wrap(err, "while executing CreateEvaluation request")
	}
	defer func() {
		if closeErr := c.closeResponseBody(response); closeErr != nil {
			err = kebError.AsTemporaryError(closeErr, "while closing CreateEvaluation response")
		}
	}()

	err = json.NewDecoder(response.Body).Decode(&responseObject)
	if err != nil {
		return nil, errors.Wrap(err, "while decode create evaluation response")
	}

	return &responseObject, nil
}

func (c *Client) RemoveReferenceFromParentEval(evaluationId int64) (err error) {
	absoluteURL := fmt.Sprintf("%s/child/%d", appendId(c.avsConfig.ApiEndpoint, c.avsConfig.ParentId), evaluationId)
	response, err := c.deleteRequest(absoluteURL)
	if err == nil {
		return nil
	}

	if response != nil && response.Body != nil {
		defer func() {
			if closeErr := c.closeResponseBody(response); closeErr != nil {
				err = kebError.AsTemporaryError(closeErr, "while closing body")
			}
		}()
		var responseObject avsNonSuccessResp
		err := json.NewDecoder(response.Body).Decode(&responseObject)
		if err != nil {
			return errors.Wrapf(err, "while decoding avs non success response body for ID: %d", evaluationId)
		}

		if strings.Contains(strings.ToLower(responseObject.Message), "does not contain subevaluation") {
			return nil
		}
	}

	return fmt.Errorf("unexpected response for evaluationId: %d while deleting reference from parent evaluation, error: %s", evaluationId, err)
}

func (c *Client) DeleteEvaluation(evaluationId int64) (err error) {
	absoluteURL := appendId(c.avsConfig.ApiEndpoint, evaluationId)
	response, err := c.deleteRequest(absoluteURL)
	defer func() {
		if closeErr := c.closeResponseBody(response); closeErr != nil {
			err = kebError.AsTemporaryError(closeErr, "while closing DeleteEvaluation response body")
		}
	}()
	if err != nil {
		return errors.Wrap(err, "while deleting evaluation")
	}

	return nil
}

func appendId(baseUrl string, id int64) string {
	if strings.HasSuffix(baseUrl, "/") {
		return baseUrl + strconv.FormatInt(id, 10)
	} else {
		return baseUrl + "/" + strconv.FormatInt(id, 10)
	}
}

func (c *Client) deleteRequest(absoluteURL string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, absoluteURL, nil)
	if err != nil {
		return &http.Response{}, errors.Wrap(err, "while creating delete request")
	}

	response, err := c.execute(req, true, true)
	if err != nil {
		return &http.Response{}, errors.Wrapf(err, "while executing delete request for path: %s", absoluteURL)
	}

	return response, nil
}

func (c *Client) execute(request *http.Request, allowNotFound bool, allowResetToken bool) (*http.Response, error) {
	httpClient, err := getHttpClient(c.ctx, c.avsConfig)
	if err != nil {
		return &http.Response{}, errors.Wrap(err, "while getting http client")
	}
	defer httpClient.CloseIdleConnections()
	response, err := httpClient.Do(request)
	if err != nil {
		return &http.Response{}, kebError.AsTemporaryError(err, "while executing request by http client")
	}

	if response.StatusCode >= http.StatusInternalServerError {
		return response, kebError.NewTemporaryError("avs server returned %d status code", response.StatusCode)
	}

	switch response.StatusCode {
	case http.StatusOK, http.StatusCreated:
		return response, nil
	case http.StatusNotFound:
		if allowNotFound {
			return response, nil
		}
		return response, fmt.Errorf("response status code: %d for %s", http.StatusNotFound, request.URL.String())
	case http.StatusUnauthorized:
		if allowResetToken {
			return c.execute(request, allowNotFound, false)
		}
		return response, fmt.Errorf("avs server returned %d status code twice for %s (response body: %s)", http.StatusUnauthorized, request.URL.String(), responseBody(response))
	}

	if response.StatusCode >= http.StatusInternalServerError {
		return response, kebError.NewTemporaryError("avs server returned %d status code", response.StatusCode)
	}

	return response, fmt.Errorf("unsupported status code: %d for %s (response body: %s)", response.StatusCode, request.URL.String(), responseBody(response))
}

func (c *Client) closeResponseBody(response *http.Response) error {
	if response == nil {
		return nil
	}
	if response.Body == nil {
		return nil
	}
	return response.Body.Close()
}

func responseBody(resp *http.Response) string {
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(bodyBytes)
}

func getHttpClient(ctx context.Context, cfg Config) (http.Client, error) {
	config := oauth2.Config{
		ClientID: cfg.OauthClientId,
		Endpoint: oauth2.Endpoint{
			TokenURL:  cfg.OauthTokenEndpoint,
			AuthStyle: oauth2.AuthStyleInHeader,
		},
	}

	initialToken, err := config.PasswordCredentialsToken(ctx, cfg.OauthUsername, cfg.OauthPassword)
	if err != nil {
		return http.Client{}, kebError.AsTemporaryError(err, "while fetching initial token")
	}

	return *config.Client(ctx, initialToken), nil
}
