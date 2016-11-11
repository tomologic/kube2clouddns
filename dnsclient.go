// Copyright 2016 Google Inc. All Rights Reserved.
// Modifications copyright (C) 2016 Tomologic
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.


package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/certifi/gocertifi"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"
)

var httpClient http.Client

func init() {
	// Use the Root Certificates bundle from the Certifi project so we don't
	// rely on the host OS or container base images for a CA Bundle.
	// See https://certifi.io for more details.
	certPool, err := gocertifi.CACerts()
	if err != nil {
		log.Fatal(err)
	}
	httpClient = http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: certPool},
		},
	}
}

type CloudDnsClient struct {
	domain  string
	project string
	*dns.Service
}

func NewDNSClient(serviceAccount []byte, domain string, project string) (*CloudDnsClient, error) {
	jwtConfig, err := google.JWTConfigFromJSON(
		serviceAccount,
		dns.NdevClouddnsReadwriteScope,
	)
	if err != nil {
		return nil, err
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &httpClient)

	jwtHTTPClient := jwtConfig.Client(ctx)
	service, err := dns.New(jwtHTTPClient)
	if err != nil {
		return nil, err
	}

	return &CloudDnsClient{domain, project, service}, nil
}

func (c *CloudDnsClient) upsert(subDomain, value string, ttl int) error {
	log.Printf("Try to upsert DNS record for %s with IP %s and TTL %s", subDomain, value, ttl)

	// Construct the new service DNS name
	recordSetName := subDomain + "." + c.domain + "."

	// Find the project dns zone first
	zone, err := c.getZoneFromProjectAndDomain()
	if err != nil {
		return err
	}

	// Do we already have this service registered? If so, delete them
	recordSetsToDelete, err := c.getRecordSetsFromName(zone, recordSetName)
	if err != nil {
		return err
	}

	record := &dns.ResourceRecordSet{
		Name:    recordSetName,
		Rrdatas: []string{value},
		Ttl:     int64(ttl),
		Type:    "A",
	}

	change := &dns.Change{
		Additions: []*dns.ResourceRecordSet{record},
		Deletions: recordSetsToDelete,
	}

	changesCreateCall, err := c.Changes.Create(c.project, zone.Name, change).Do()
	if err != nil {
		return err
	}

	for changesCreateCall.Status == "pending" {
		time.Sleep(time.Second)
		changesCreateCall, err = c.Changes.Get(c.project, zone.Name, changesCreateCall.Id).Do()
		if err != nil {
			return err
		}
	}

	log.Printf("Upsert completed")

	return nil
}

func (c *CloudDnsClient) delete(subDomain string) error {

	recordSetName := subDomain + "." + c.domain + "."

	zone, err := c.getZoneFromProjectAndDomain()
	if err != nil {
		return err
	}

	matchingRecords, err := c.getRecordSetsFromName(zone, recordSetName)
	if err != nil {
		return err
	}

	change := &dns.Change{
		Deletions: matchingRecords,
	}
	changesCreateCall, err := c.Changes.Create(c.project, zone.Name, change).Do()
	if err != nil {
		return err
	}
	for changesCreateCall.Status == "pending" {
		time.Sleep(time.Second)
		changesCreateCall, err = c.Changes.Get(c.project, zone.Name, changesCreateCall.Id).Do()
		if err != nil {
			return err
		}
	}
	log.Printf("Record %s deleted", subDomain)
	return nil
}

func (c *CloudDnsClient) getZoneFromProjectAndDomain() (*dns.ManagedZone, error) {
	// Get list of zones from the project in order to find a zone with the selected domain
	zones, err := c.ManagedZones.List(c.project).Do()
	if err != nil {
		return nil, err
	}

	var zone *dns.ManagedZone
	for _, zoneInEvaluation := range zones.ManagedZones {
		if strings.HasSuffix(c.domain + ".", zoneInEvaluation.DnsName) {
			zone = zoneInEvaluation
		}
	}
	if zone == nil {
		return nil, errors.New("Zone matching the domain not found")
	}
	return zone, nil
}

func (c *CloudDnsClient) getRecordSetsFromName(zone *dns.ManagedZone, recordName string) ([]*dns.ResourceRecordSet, error) {
	matchingRecords := []*dns.ResourceRecordSet{}

	records, err := c.ResourceRecordSets.List(c.project, zone.Name).Do()
	if err != nil {
		return matchingRecords, err
	}

	for _, record := range records.Rrsets {
		if record.Type == "A" && record.Name == recordName {
			matchingRecords = append(matchingRecords, record)
		}
	}
	return matchingRecords, nil
}
