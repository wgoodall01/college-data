package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

type CollegeBase struct {
	// ApiKey is the API key for the base
	ApiKey string

	// BaseID is the airtable base ID
	BaseID string

	// TableName is the name of the table which data will be fetched for
	TableName string

	// rateLimit is a ticker limiting requests to 5/second.
	limiter *time.Ticker
}

func (cb *CollegeBase) rootUrl() string {
	return fmt.Sprintf("https://api.airtable.com/v0/%s/%s", cb.BaseID, cb.TableName)
}

func (cb *CollegeBase) authorizeReq(req *http.Request) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cb.ApiKey))
}

func (cb *CollegeBase) waitForLimiter() {
	if cb.limiter == nil {
		cb.limiter = time.NewTicker(time.Second / 5.0) // 5 req/sec
	} else {
		<-cb.limiter.C // wait for the 0.2s.
	}

}

type airtableError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (cb *CollegeBase) Patch(c *College) error {
	// serialize the college
	apiReq := struct {
		Fields *College `json:"fields"`
	}{
		Fields: c,
	}
	jsonBytes, jsonErr := json.Marshal(&apiReq)
	if jsonErr != nil {
		return jsonErr
	}
	jsonBuf := bytes.NewBuffer(jsonBytes)

	patchUrl := fmt.Sprintf("%s/%s", cb.rootUrl(), c.ID)
	req, reqErr := http.NewRequest("PATCH", patchUrl, jsonBuf)
	if reqErr != nil {
		return reqErr
	}
	cb.authorizeReq(req)
	req.Header.Add("Content-Type", "application/json")

	// Send the request
	cb.waitForLimiter()
	resp, fetchErr := http.DefaultClient.Do(req)
	if fetchErr != nil {
		return fetchErr
	}

	if resp.StatusCode != 200 {
		body, readErr := ioutil.ReadAll(resp.Body)
		if readErr != nil {
			return errors.Wrap(readErr, "err reading error:")
		}

		var errResp struct {
			Error airtableError
		}
		decodeErr := json.Unmarshal(body, &errResp)
		if decodeErr != nil {
			return errors.Wrap(decodeErr, "error decoding error:")
		}

		errInfo := errResp.Error
		return errors.Errorf("airtable: %s: %s", errInfo.Type, errInfo.Message)
	}

	return nil
}

func (cb *CollegeBase) Colleges() ([]*College, error) {
	req, reqErr := http.NewRequest("GET", cb.rootUrl(), nil)
	if reqErr != nil {
		return nil, reqErr
	}

	query := req.URL.Query()
	query.Add("filterByFormula", "NOT({_big_future_id} = '')")
	req.URL.RawQuery = query.Encode()

	cb.authorizeReq(req)

	cb.waitForLimiter()
	resp, fetchErr := http.DefaultClient.Do(req)
	if fetchErr != nil {
		return nil, fetchErr
	}

	buf, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, readErr
	}

	var apiResp struct {
		Records []struct {
			ID     string
			Fields College
		}
		Error *airtableError
	}
	decodeErr := json.Unmarshal(buf, &apiResp)
	if decodeErr != nil {
		return nil, decodeErr
	}

	errInfo := apiResp.Error
	if errInfo != nil {
		return nil, errors.Errorf("airtable: %s: %s", errInfo.Type, errInfo.Message)
	}

	// add the ID to each college record
	colleges := make([]*College, len(apiResp.Records))
	for i, record := range apiResp.Records {
		college := record.Fields
		college.ID = record.ID
		colleges[i] = &college
	}

	return colleges, nil
}
