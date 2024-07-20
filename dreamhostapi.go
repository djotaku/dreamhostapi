// Package dreamhostapi contains functions for interacting with the Dreamhost API.
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

// A DnsRecordsJSON holds the data field of the JSON returned by the Dreamhost API when given the command dns-list_records.
type DnsRecordsJSON struct {
	Data []map[string]string `json:"data"` // A slice of maps representing the key/pair values for the DNS records.
}

// A commandResult holds the JSON result from the Dreamhost API.
// In this package, it's mostly used to return whether a command generated a successful response from the API.
type commandResult struct {
	Data   string `json:"data"`   // A string that would need to be unmarshalled to turn it into a map.
	Result string `json:"result"` // A string representing whether the API was successfully accessed or not.
}

// webGet gets the data from a url.
// It returns the body as a string, an int representing the HTTP status code, and any errors.
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

// submitDreamhostCommand returns the response from the Dreamhost API as JSON as well as any errors.
// At this stage the JSON is not unmarshalled, it is returned as a string.
func submitDreamhostCommand(command map[string]string, apiKey string) (string, error) {
	apiURLBase := "https://api.dreamhost.com/?"
	queryParameters := url.Values{}
	queryParameters.Set("key", apiKey)
	for key, value := range command {
		queryParameters.Add(key, value)
	}
	queryParameters.Add("format", "json")
	fullURL := apiURLBase + queryParameters.Encode()
	dreamhostResponse, statusCode, err := WebGet(fullURL)
	if err != nil {
		return dreamhostResponse, err
	}
	if statusCode == 429 {
		fmt.Println("Rate limit hit. Pausing execution for 10 minutes.")
		time.Sleep(600 * time.Second)
		dreamhostResponse, err = submitDreamhostCommand(command, apiKey)
	}
	return dreamhostResponse, err
}

// getDNSRecords returns the unmarshalled JSON response containing all of the DNS records that correspond to this apiKey and any errors.
func GetDNSRecords(apiKey string) (string, error) {
	command := map[string]string{"cmd": "dns-list_records"}
	dnsRecords, err := submitDreamhostCommand(command, apiKey)
	if err != nil {
		return "", err
	}
	return dnsRecords, err
}

// addDNSRecord returns the JSON "result" field after using the Dreamhost API to add an IP address to a domain in dreamhost and any errors.
func AddDNSRecord(domain string, newIPAddress string, apiKey string) (string, error) {
	command := map[string]string{"cmd": "dns-add_record", "record": domain, "type": "A", "value": newIPAddress}
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

// deleteDNSRecord returns the JSON "result" field after using the Dreamhost API to delete an IP address from a domain in dreamhost and any errors.
func DeleteDNSRecord(domain string, newIPAddress string, apiKey string) (string, error) {
	command := map[string]string{"cmd": "dns-remove_record", "record": domain, "type": "A", "value": newIPAddress}
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

// updateDNSRecord returns the JSON "result" field after using the Dreamhost API to first add the new IP address and, if successful, deleting the old one.
// At whatever stage it errors out, it returns the empty string. So 2 empty strings would mean both operations errored.
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
