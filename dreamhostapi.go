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
func WebGet(url string) string {
	response, err := http.Get(url)
	if err != nil {
		log.Println(err)
	}
	result, err := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode > 299 {
		statusCodeString := fmt.Sprintf("Response failed with status code: %d and \nbody: %s\n", response.StatusCode, result)
		log.Println(statusCodeString)
	}
	if err != nil {
		log.Println(err)
	}
	return string(result)
}

//submitDreamhostCommand takes in a command string and api key, contacts the API and returns the result
func submitDreamhostCommand(command string, apiKey string) string {
	apiURLBase := "https://api.dreamhost.com/?"
	queryParameters := url.Values{}
	queryParameters.Set("key", apiKey)
	queryParameters.Add("cmd", command)
	queryParameters.Add("format", "json")
	fullURL := apiURLBase + queryParameters.Encode()
	dreamhostResponse := WebGet(fullURL)
	return dreamhostResponse
}

//getDNSRecords gets the DNS records from the Dreamhost API
func GetDNSRecords(apiKey string) string {
	dnsRecords := submitDreamhostCommand("dns-list_records", apiKey)
	return dnsRecords
}

//conditionalLog will print a log to the console if logActive true
func conditionalLog(message string, logActive bool) {
	if logActive {
		log.Println(message)
	}
}

// addDNSRecord adds an IP address to a domain in dreamhost
func AddDNSRecord(domain string, newIPAddress string, apiKey string) string {
	command := "dns-add_record&record=" + domain + "&type=A" + "&value=" + newIPAddress
	response := submitDreamhostCommand(command, apiKey)
	var result commandResult
	err := json.Unmarshal([]byte(response), &result)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}
	log.Printf("Result of trying to add DNS record for %s is %s\n", domain, result.Data)
	return result.Data
}

// deleteDNSRecord deletes an IP address to a domain in dreamhost
func DeleteDNSRecord(domain string, newIPAddress string, apiKey string) string {
	command := "dns-remove_record&record=" + domain + "&type=A" + "&value=" + newIPAddress
	response := submitDreamhostCommand(command, apiKey)
	var result commandResult
	err := json.Unmarshal([]byte(response), &result)
	if err != nil {
		log.Printf("Error: %s\n", err)
	}
	log.Printf("Result of trying to delete DNS record for %s is %s\n", domain, result.Data)
	return result.Data
}

//updateDNSRecord adds a record and, if successful, deletes the old one.
func UpdateDNSRecord(domain string, currentIP string, newIPAddress string, apiKey string) {
	resultOfAdd := AddDNSRecord(domain, newIPAddress, apiKey)
	if resultOfAdd == "sucess" {
		DeleteDNSRecord(domain, currentIP, apiKey)
	}
}
