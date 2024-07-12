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
	Data []DnsRecord `json:"data"`
}

// DnsRecord is a DNS Record on Dreamhost
type DnsRecord struct {
	Record    string // the URL
	Zone      string // This is the base of the URL. If Record is www.google.com, Zone is google.com
	Value     string // this is what the zone points to - usually IP address
	Editable  string // 0 or 1 value, but comes back as a string
	ZoneType  string `json:"type"` // zone type, eg A, AAA, MX, etc
	Comment   string // comment that can be added to a record
	AccountId string `json:"account_id"` // the account associated with this record
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
		dreamhostResponse, statusCode, err = WebGet(fullURL)
	}
	return dreamhostResponse, err
}

// getDNSRecords gets the DNS records from the Dreamhost API
func GetDNSRecords(apiKey string) (DnsRecordsJSON, error) {
	command := map[string]string{"cmd": "dns-list_records"}
	dnsRecords, err := submitDreamhostCommand(command, apiKey)
	if err != nil {
		var emptyrecords DnsRecordsJSON
		return emptyrecords, err
	}
	var records DnsRecordsJSON
	err = json.Unmarshal([]byte(dnsRecords), &records)
	if err != nil {
		var emptyrecords DnsRecordsJSON
		return emptyrecords, err
	}
	return records, err
}

// addDNSRecord adds an IP address to a domain in dreamhost
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

// deleteDNSRecord deletes an IP address to a domain in dreamhost
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
