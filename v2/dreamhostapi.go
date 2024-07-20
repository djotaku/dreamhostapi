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

// dnsRecords holds an array of DnsRecord structs returned by the Dreamhost API
type DnsRecords struct {
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

func (r DnsRecord) String() string {
	return fmt.Sprintf("\nRecord (URL): %s in Zone: %s. \nIt points to %s. \nZone Type: %s \nIs it Editable? %s. \nIt Belongs to: %s. \nComment: %s\n", r.Record, r.Zone, r.Value, r.ZoneType, r.Editable, r.AccountId, r.Comment)
}

// A commandResult holds the JSON result from the Dreamhost API.
// In this package, it's mostly used to return whether a command generated a successful response from the API.
type commandResult struct {
	Data   string `json:"data"`
	Result string `json:"result"`
}

// webGet returns the body as a string, an int representing the HTTP status code, and any errors.
func webGet(url string) (string, int, error) {
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
// The command map is essentially a map in which the keys correspond to the editable fields in the DNS Record and the fields are the values to change.
// See the add and return commands in this package for examples.
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
	dreamhostResponse, statusCode, err := webGet(fullURL)
	if err != nil {
		return dreamhostResponse, err
	}
	if statusCode == 429 {
		fmt.Println("Rate limit hit. Pausing execution for 10 minutes.")
		time.Sleep(600 * time.Second)
		submitDreamhostCommand(command, apiKey)
	}
	return dreamhostResponse, err
}

// getDNSRecords returns a DnsRecords struct containing all of the DNS records that correspond to this apiKey and any errors.
func GetDNSRecords(apiKey string) (DnsRecords, error) {
	command := map[string]string{"cmd": "dns-list_records"}
	dnsRecords, err := submitDreamhostCommand(command, apiKey)
	if err != nil {
		var emptyrecords DnsRecords
		return emptyrecords, err
	}
	var records DnsRecords
	err = json.Unmarshal([]byte(dnsRecords), &records)
	if err != nil {
		var emptyrecords DnsRecords
		return emptyrecords, err
	}
	return records, err
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
