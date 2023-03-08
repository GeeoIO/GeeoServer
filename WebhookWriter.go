package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

// HookMessage is the format of messages sent to hooks
type HookMessage struct {
	// AirBeacon is the beacon concerned by the message
	AirBeacon string `json:"beacon_id"`
	// Message is the actual message sent
	Message interface{} `json:"message"`
}

// WebhookWriter regularly sends updates to a webhook with a POST http request
type WebhookWriter struct {
	sync.Mutex
	baseurl     string
	bearerToken string
	quit        chan (bool)
	headers     map[string]string
	messages    []HookMessage
}

// NewWebhookWriter creates and returns a new WebhookWriter
// it starts a goroutine to send messages every MessageSendInterval ms.
func NewWebhookWriter(url string, headers map[string]string, bearerToken string) *WebhookWriter {
	whw := &WebhookWriter{baseurl: url, headers: headers, bearerToken: bearerToken}

	go func() {
		ticker := time.NewTicker(MessageSendInterval)
		for {
			select {
			case <-ticker.C:
				whw.send()

			case <-whw.quit:
				ticker.Stop()
			}
		}
	}()

	return whw
}

func (whw *WebhookWriter) Write(beacon *AirBeacon, message HookMessage) {
	whw.Lock()
	defer whw.Unlock()

	whw.messages = append(whw.messages, message)
	log.Debug("Adding one message to WHW: ", message)
}

func (whw *WebhookWriter) send() {
	if len(whw.messages) == 0 {
		return
	}
	log.Debug("Sending WHW messages")
	whw.Lock()
	defer whw.Unlock()

	b, err := json.Marshal(whw.messages)
	if err != nil {
		log.Error(err)
	}
	whw.messages = nil

	go func() {
		tr := &http.Transport{DisableKeepAlives: true}
		hc := http.Client{Transport: tr}

		req, err := http.NewRequest("POST", whw.baseurl, bytes.NewReader([]byte(b)))

		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Authorization", "Bearer "+whw.bearerToken)
		req.Header.Set("User-Agent", "Geeo.io webhook handler")

		for h, v := range whw.headers {
			req.Header.Add(h, v)
		}

		resp, err := hc.Do(req)
		if resp != nil {
			defer resp.Body.Close()
			_, err = io.Copy(ioutil.Discard, resp.Body)
			log.Debugf("Webhook replied with status code %d", resp.StatusCode)
		}

		if err != nil {
			log.Warn("Webhook POST error: ", err)
			return
		}
	}()
}
