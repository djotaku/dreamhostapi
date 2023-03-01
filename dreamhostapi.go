//Package dreamhostapi contains functions for interacting with the Dreamhost API
package dreamhostapi

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

//dnsRecordsJSON holds the JSON returned by the Dreamhost API
type DnsRecordsJSON struct {
	Data []map[string]string `json:"data"`
}

// commandResult for when you only care about the result
type commandResult struct {
	Data string `json:"result"`
}

// webGet handles contacting a URL
func WebGet(url string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "Error accessing URL", err
	}
	result, err := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode > 299 {
		statusCodeString := fmt.Sprintf("Response failed with status code: %d and \nbody: %s\n", response.StatusCode, result)
		log.Println(statusCodeString)
	}
	if err != nil {
		return "Error reading response", err
	}
	return string(result), err
}

//submitDreamhostCommand takes in a command string and api key, contacts the API and returns the result
func submitDreamhostCommand(command string, apiKey string) (string, error) {
	apiURLBase := "https://api.dreamhost.com/?"
	queryParameters := url.Values{}
	queryParameters.Set("key", apiKey)
	queryParameters.Add("cmd", command)
	queryParameters.Add("format", "json")
	fullURL := apiURLBase + queryParameters.Encode()
	dreamhostResponse, err := WebGet(fullURL)
	if err != nil {
		return "", err
	}
	return dreamhostResponse, err
}

//getDNSRecords gets the DNS records from the Dreamhost API
func GetDNSRecords(apiKey string) (string, error) {
	dnsRecords, err := submitDreamhostCommand("dns-list_records", apiKey)
	if err != nil {
		return "", err
	}
	return dnsRecords, err
}

//conditionalLog will print a log to the console if logActive true
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
	if err != nil {
		return "", jsonErr
	}
	return result.Data, err
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
	log.Printf("Result of trying to delete DNS record for %s is %s\n", domain, result.Data)
	return result.Data, jsonErr
}

//updateDNSRecord adds a record and, if successful, deletes the old one.
func UpdateDNSRecord(domain string, currentIP string, newIPAddress string, apiKey string) (string, string, error) {
	resultOfAdd, err := AddDNSRecord(domain, newIPAddress, apiKey)
	if err != nil {
		return "", "", err
	}
	resultOfDelete := ""
	if resultOfAdd == "sucess" {
		resultOfDelete, err = DeleteDNSRecord(domain, currentIP, apiKey)
		if err != nil {
			return resultOfAdd, "", err
		}
	}
	return resultOfAdd, resultOfDelete, err
}
