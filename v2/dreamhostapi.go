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
	Data   []DnsRecord `json:"data"`
	Result string      `json:"result"`
}

// DnsRecord is a DNS Record on Dreamhost
type DnsRecord struct {
	Record    string // the URL
	Zone      string // This is the base of the URL. If Record is www.google.com, Zone is google.com
	Value     string // this is what the zone points to - usually IP address
	Editable  string // 0 or 1 value, but comes back as a string
	ZoneType  string `json:"type"` // zone type: A,CNAME,NS,NAPTR,SRV,TXT, or AAAA
	Comment   string // comment that can be added to a record
	AccountId string `json:"account_id"` // the account associated with this record
}

func (r DnsRecord) String() string {
	return fmt.Sprintf("\nRecord (URL): %s in Zone: %s. \nIt points to %s. \nZone Type: %s \nIs it Editable? %s. \nIt Belongs to: %s. \nComment: %s\n", r.Record, r.Zone, r.Value, r.ZoneType, r.Editable, r.AccountId, r.Comment)
}

// webGet returns the body as a string, an int representing the HTTP status code, and any errors.
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
// In the case of any errors (eg web or JSON unmarshalling) it returns an empty struct.
// The command map is essentially a map in which the keys correspond to the editable fields in the DNS Record and the fields are the values to change.
// As of now, all [Dreamhost DNS commands] are implemented.
//
// [Dreamhost DNS commands]: https://help.dreamhost.com/hc/en-us/articles/217555707-DNS-API-commands
func submitDreamhostCommand(command map[string]string, apiKey string) (DnsRecords, error) {
	var records DnsRecords
	var emptyrecords DnsRecords // used for times when errors mean we're returning an empty DNS Records struct
	apiURLBase := "https://api.dreamhost.com/?"
	queryParameters := url.Values{}
	queryParameters.Set("key", apiKey)
	for key, value := range command {
		queryParameters.Add(key, value)
	}
	queryParameters.Add("format", "json")
	fullURL := apiURLBase + queryParameters.Encode()
	dreamhostResponse, statusCode, err := webGet(fullURL)
	if err != nil { // there was an error at the web level.
		return emptyrecords, err
	}
	err = json.Unmarshal([]byte(dreamhostResponse), &records)
	if err != nil {
		return emptyrecords, err // there was an error at the JSON unmarshalling level
	}
	if statusCode == 429 {
		fmt.Println("Rate limit hit. Pausing execution for 10 minutes.")
		time.Sleep(600 * time.Second)
		records, err = submitDreamhostCommand(command, apiKey)
	}
	return records, err
}

// getDNSRecords returns a DnsRecords struct containing all of the DNS records that correspond to this apiKey and any errors.
// It returns an empty struct in the case of any errors in the web-layer, JSON demarshalling, or API non-success result.
func GetDNSRecords(apiKey string) (DnsRecords, error) {
	command := map[string]string{"cmd": "dns-list_records"}
	dnsRecords, err := submitDreamhostCommand(command, apiKey)
	if err != nil {
		return dnsRecords, err // will already be the empty record
	}
	if dnsRecords.Result != "success" { // we hit the API successfully, but did not get back JSON successfully. eg: bad APIKey.
		var emptyrecords DnsRecords
		return emptyrecords, err
	}
	return dnsRecords, err
}

// UpdateZoneFile returns a DnsRecords after using the Dreamhost API to either add or delete an IP address from a domain in Dreamhost and any errors.
// In the case of a success, it should only contain one record in the slice.
// It returns an empty struct in the case of any errors in the web-layer, JSON demarshalling, or API non-success result.
// Currently implemented commands for the command parameter are:
//   - "add" to add a value (typically IP address) to a record (typically a domain).
//   - "del" to remove a value (typically IP address) from a record (typically a domain).
func UpdateZoneFIle(command string, domain string, IPAddress string, apiKey string, comment string) (DnsRecords, error) {
	var commandOptions map[string]string
	switch command {
	case "add":
		commandOptions = map[string]string{"cmd": "dns-add_record", "record": domain, "type": "A", "value": IPAddress, "comment": comment}
	case "del":
		commandOptions = map[string]string{"cmd": "dns-remove_record", "record": domain, "type": "A", "value": IPAddress, "comment": comment}
	}
	response, err := submitDreamhostCommand(commandOptions, apiKey)
	if err != nil {
		return response, err
	}
	return response, err
}

// updateDNSRecord returns the JSON "result" field after using the Dreamhost API to first add the new IP address and, if successful, deleting the old one.
// At whatever stage it errors out, it returns the empty string. So 2 empty strings would mean both operations errored.
func UpdateDNSRecord(domain string, currentIP string, newIPAddress string, apiKey string, comment string) (DnsRecords, DnsRecords, error) {
	resultOfAdd, err := UpdateZoneFIle("add", domain, newIPAddress, apiKey, comment)
	var emptyrecords DnsRecords
	if err != nil {
		return emptyrecords, emptyrecords, err
	}
	resultOfDelete, err := UpdateZoneFIle("del", domain, currentIP, apiKey, comment)
	if err != nil {
		return resultOfAdd, resultOfDelete, err
	}
	return resultOfAdd, resultOfDelete, err
}
