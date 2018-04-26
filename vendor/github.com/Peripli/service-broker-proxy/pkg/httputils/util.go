package httputils

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// GetContent of the request inside given struct
func GetContent(v interface{}, closer io.ReadCloser) error {
	body, err := ioutil.ReadAll(closer)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, v)
	if err != nil {
		return err
	}
	return nil
}

func SendRequest(client *http.Client, method, URL string, params map[string]string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}
	request, err := http.NewRequest(method, URL, bodyReader)

	if err != nil {
		return nil, err
	}

	if params != nil {
		q := request.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		request.URL.RawQuery = q.Encode()
	}

	return client.Do(request)
}

func HandleResponseError(response *http.Response) error {
	logrus.Info("handling failure responses")

	httpErr := HTTPErrorResponse{
		StatusCode: response.StatusCode,
	}

	brokerResponse := make(map[string]interface{})
	if err := GetContent(&brokerResponse, response.Body); err != nil {
		httpErr.ErrorMessage = err.Error()
		return errors.Wrap(err, "error handling failure response")
	}

	if errorKey, ok := brokerResponse["error"].(string); ok {
		httpErr.ErrorKey = errorKey
	}

	if description, ok := brokerResponse["description"].(string); ok {
		httpErr.ErrorMessage = description
	}

	return httpErr
}

func WriteResponse(w http.ResponseWriter, code int, object interface{}) {
	data, err := json.Marshal(object)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(code)
	w.Write(data)
}
