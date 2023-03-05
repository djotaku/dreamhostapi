// Package dreamhostapi contains functions for interacting with the Dreamhost API
package dreamhostapi

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type DreamhostAPIError string

func (apiErr DreamhostAPIError) Error() string {
	return string(apiErr)
}

// dnsRecordsJSON holds the JSON returned by the Dreamhost API
type DnsRecordsJSON struct {
	Data []map[string]string `json:"data"`
}

// commandResult for when you only care about the result
type commandResult struct {
	Data   string `json:"data"`
	Result string `json:"result"`
}

// webGet handles contacting a URL
func WebGet(url string) (string, int, error) {
	response, err := http.Get(url)
	if err != nil {
		return "Error accessing URL", 0, err
	}
	result, err := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode > 299 {
		statusCodeString := fmt.Sprintf("Response failed with status code: %d and \nbody: %s\n", response.StatusCode, result)
		log.Println(statusCodeString)
	}
	if err != nil {
		return "Error reading response", 0, err
	}
	return string(result), response.StatusCode, err
}

// submitDreamhostCommand takes in a command string and api key, contacts the API and returns the result
func submitDreamhostCommand(command string, apiKey string) (string, error) {
	apiURLBase := "https://api.dreamhost.com/?"
	queryParameters := url.Values{}
	queryParameters.Set("key", apiKey)
	queryParameters.Add("cmd", command)
	queryParameters.Add("format", "json")
	fullURL := apiURLBase + queryParameters.Encode()
  // debug
  fmt.Println(fullURL)
	dreamhostResponse, statusCode, err := WebGet(fullURL)
	if err != nil {
		return dreamhostResponse, err
	}
	if statusCode == 429 {
		fmt.Println("Rate limit hit. Pausing execution for 10 minutes.")
		time.Sleep(600 * time.Second)
		dreamhostResponse, statusCode, err = WebGet(fullURL)
	}
	return dreamhostResponse, err
}

// getDNSRecords gets the DNS records from the Dreamhost API
func GetDNSRecords(apiKey string) (string, error) {
	dnsRecords, err := submitDreamhostCommand("dns-list_records", apiKey)
	if err != nil {
		return "", err
	}
	return dnsRecords, err
}

// conditionalLog will print a log to the console if logActive true
func conditionalLog(message string, logActive bool) {
	if logActive {
		log.Println(message)
	}
}

// addDNSRecord adds an IP address to a domain in dreamhost
func AddDNSRecord(domain string, newIPAddress string, apiKey string) (string, error) {
	command := "dns-add_record&record=" + domain + "&type=A" + "&value=" + newIPAddress
	response, err := submitDreamhostCommand(command, apiKey)
	if err != nil {
		return "", err
	}
	var result commandResult
	jsonErr := json.Unmarshal([]byte(response), &result)
	if jsonErr != nil {
		return "", jsonErr
	}
	if result.Result == "error" {
		err = DreamhostAPIError(result.Data)
	}
	return result.Result, err
}

// deleteDNSRecord deletes an IP address to a domain in dreamhost
func DeleteDNSRecord(domain string, newIPAddress string, apiKey string) (string, error) {
	command := "dns-remove_record&record=" + domain + "&type=A" + "&value=" + newIPAddress
	response, err := submitDreamhostCommand(command, apiKey)
	if err != nil {
		return "", err
	}
	var result commandResult
	jsonErr := json.Unmarshal([]byte(response), &result)
	if jsonErr != nil {
		return "", jsonErr
	}
	if result.Result == "error" {
		err = DreamhostAPIError(result.Data)
	}
	return result.Result, jsonErr
}

// updateDNSRecord adds a record and, if successful, deletes the old one.
func UpdateDNSRecord(domain string, currentIP string, newIPAddress string, apiKey string) (string, string, error) {
	resultOfAdd, err := AddDNSRecord(domain, newIPAddress, apiKey)
	if err != nil {
		return "", "", err
	}
	resultOfDelete, err := DeleteDNSRecord(domain, currentIP, apiKey)
	if err != nil {
		return resultOfAdd, "", err
	}
	return resultOfAdd, resultOfDelete, err
}
