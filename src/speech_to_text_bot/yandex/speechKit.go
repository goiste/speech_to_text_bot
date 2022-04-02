package yandex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	recognizeURL   = "https://transcribe.api.cloud.yandex.net/speech/stt/v2/longRunningRecognize"
	checkResultURL = "https://operation.api.cloud.yandex.net/operations/"
)

type RecognizeResult struct {
	Done     bool `json:"done"`
	Response struct {
		Type   string `json:"@type"`
		Chunks []struct {
			Alternatives []struct {
				Words []struct {
					StartTime  string `json:"startTime"`
					EndTime    string `json:"endTime"`
					Word       string `json:"word"`
					Confidence int    `json:"confidence"`
				} `json:"words"`
				Text       string `json:"text"`
				Confidence int    `json:"confidence"`
			} `json:"alternatives"`
			ChannelTag string `json:"channelTag"`
		} `json:"chunks"`
	} `json:"response"`
	Id         string    `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	CreatedBy  string    `json:"createdBy"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

type SendRecognizeRequestResult struct {
	Done       bool      `json:"done"`
	Id         string    `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	CreatedBy  string    `json:"createdBy"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

func (c *YClient) Recognize(fileName string, audioContent []byte) (string, error) {
	err := c.UploadFile(fileName, audioContent)
	if err != nil {
		return "", fmt.Errorf("upload file error: %w", err)
	}

	sendRecResult, err := c.SendToRecognize(fileName)
	if err != nil {
		return "", fmt.Errorf("send to recognize error: %w", err)
	}

	done := sendRecResult.Done
	var recResult *RecognizeResult
	for !done {
		recResult, err = c.GetRecognizeResult(sendRecResult.Id)
		if err != nil {
			return "", fmt.Errorf("check recognize result error: %w", err)
		}
		done = recResult.Done
		if !done {
			time.Sleep(time.Second)
		}
	}

	text := getText(recResult)

	err = c.DeleteFile(fileName)
	if err != nil {
		err = fmt.Errorf("delete file error: %w", err)
	}

	return text, err
}

func (c *YClient) SendToRecognize(key string) (*SendRecognizeRequestResult, error) {
	jsonStr, _ := json.Marshal(map[string]interface{}{
		"config": map[string]interface{}{
			"specification": map[string]interface{}{
				"literature_text": true,
			},
		},
		"audio": map[string]string{
			"uri": fmt.Sprintf("%s/%s/%s", storageEndpoint, c.conf.AwsBucket, key),
		},
	})

	req, err := http.NewRequest(http.MethodPost, recognizeURL, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Api-Key "+c.conf.AwsApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request error: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body error: %w", err)
	}

	var result SendRecognizeRequestResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		err = fmt.Errorf("unmarshal result error: %w", err)
	}

	return &result, err
}

func (c *YClient) GetRecognizeResult(id string) (*RecognizeResult, error) {
	req, err := http.NewRequest(http.MethodGet, checkResultURL+id, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Api-Key "+c.conf.AwsApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request error: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body error: %w", err)
	}

	var result RecognizeResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		if len(body) == 0 {
			err = nil
			result.Done = true
		} else {
			err = fmt.Errorf("unmarshal result error: %w", err)
		}
	}

	return &result, err
}

func getText(result *RecognizeResult) string {
	if result == nil {
		return ""
	}

	text := ""

	for _, c := range result.Response.Chunks {
		text += c.Alternatives[0].Text
		text += " "
	}

	return text
}
