/*
Copyright 2022 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package porkbun

import (
	"context"
	"log/slog"
	"testing"

	pb "github.com/nrdcg/porkbun"
	"github.com/prometheus/common/promslog"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
)

func TestPorkbunProvider(t *testing.T) {
	t.Run("EndpointZoneName", testEndpointZoneName)
	t.Run("GetIDforRecord", testGetIDforRecord)
	t.Run("ConvertToPorkbunRecord", testConvertToPorkbunRecord)
	t.Run("NewPorkbunProvider", testNewPorkbunProvider)
	t.Run("ApplyChanges", testApplyChanges)
	t.Run("Records", testRecords)
}

func testEndpointZoneName(t *testing.T) {
	zoneList := []string{"bar.org", "baz.org"}

	// in zone list
	ep1 := endpoint.Endpoint{
		DNSName:    "foo.bar.org",
		Targets:    endpoint.Targets{"5.5.5.5"},
		RecordType: endpoint.RecordTypeA,
	}

	// not in zone list
	ep2 := endpoint.Endpoint{
		DNSName:    "foo.foo.org",
		Targets:    endpoint.Targets{"5.5.5.5"},
		RecordType: endpoint.RecordTypeA,
	}

	// matches zone exactly
	ep3 := endpoint.Endpoint{
		DNSName:    "baz.org",
		Targets:    endpoint.Targets{"5.5.5.5"},
		RecordType: endpoint.RecordTypeA,
	}

	assert.Equal(t, endpointZoneName(&ep1, zoneList), "bar.org")
	assert.Equal(t, endpointZoneName(&ep2, zoneList), "")
	assert.Equal(t, endpointZoneName(&ep3, zoneList), "baz.org")
}

func testGetIDforRecord(t *testing.T) {

	recordName := "foo.example.com"
	target1 := "heritage=external-dns,external-dns/owner=default,external-dns/resource=service/default/nginx"
	target2 := "5.5.5.5"
	recordType := "TXT"

	pb1 := pb.Record{
		Name:    "foo.example.com",
		Type:    "TXT",
		Content: "heritage=external-dns,external-dns/owner=default,external-dns/resource=service/default/nginx",
		ID:      "10",
	}
	pb2 := pb.Record{
		Name:    "foo.foo.org",
		Type:    "A",
		Content: "5.5.5.5",
		ID:      "10",
	}

	pb3 := pb.Record{
		ID:      "",
		Name:    "baz.org",
		Type:    "A",
		Content: "5.5.5.5",
	}

	pbRecordList := []pb.Record{pb1, pb2, pb3}

	assert.Equal(t, "10", getIDforRecord(recordName, target1, recordType, &pbRecordList))
	assert.Equal(t, "", getIDforRecord(recordName, target2, recordType, &pbRecordList))

}

func testConvertToPorkbunRecord(t *testing.T) {
	// in zone list
	ep1 := endpoint.Endpoint{
		DNSName:    "foo.bar.org",
		Targets:    endpoint.Targets{"5.5.5.5"},
		RecordType: endpoint.RecordTypeA,
	}

	// not in zone list
	ep2 := endpoint.Endpoint{
		DNSName:    "foo.foo.org",
		Targets:    endpoint.Targets{"5.5.5.5"},
		RecordType: endpoint.RecordTypeA,
	}

	// matches zone exactly
	ep3 := endpoint.Endpoint{
		DNSName:    "bar.org",
		Targets:    endpoint.Targets{"5.5.5.5"},
		RecordType: endpoint.RecordTypeA,
	}

	ep4 := endpoint.Endpoint{
		DNSName:    "foo.baz.org",
		Targets:    endpoint.Targets{"\"heritage=external-dns,external-dns/owner=default,external-dns/resource=service/default/nginx\""},
		RecordType: endpoint.RecordTypeTXT,
	}

	epList := []*endpoint.Endpoint{&ep1, &ep2, &ep3, &ep4}

	pb1Retrieved := pb.Record{
		Name:    "foo.bar.org",
		Type:    "A",
		Content: "5.5.5.5",
		ID:      "10",
	}
	pb1 := pb.Record{
		Name:    "foo",
		Type:    "A",
		Content: "5.5.5.5",
		ID:      "10",
	}
	pb2 := pb.Record{
		Name:    "foo.foo.org",
		Type:    "A",
		Content: "5.5.5.5",
		ID:      "15",
	}
	pb3retrieved := pb.Record{
		ID:      "1",
		Name:    "bar.org",
		Type:    "A",
		Content: "5.5.5.5",
	}
	pb3 := pb.Record{
		ID:      "1",
		Name:    "",
		Type:    "A",
		Content: "5.5.5.5",
	}
	pb4 := pb.Record{
		ID:      "",
		Name:    "foo.baz.org",
		Type:    "TXT",
		Content: "heritage=external-dns,external-dns/owner=default,external-dns/resource=service/default/nginx",
	}

	// The retrieved records include the zone
	pbRetrievedRecordList := []pb.Record{pb1Retrieved, pb2, pb3retrieved, pb4}
	// The records we want to create should not include the zone
	pbRecordList := []pb.Record{pb1, pb2, pb3, pb4}

	assert.Equal(t, convertToPorkbunRecord(&pbRetrievedRecordList, epList, "bar.org", false), &pbRecordList)
}

func testNewPorkbunProvider(t *testing.T) {
	domainFilter := []string{"example.com"}
	var logger *slog.Logger
	promslogConfig := &promslog.Config{}
	logger = promslog.New(promslogConfig)

	p, err := NewPorkbunProvider(&domainFilter, "KEY", "PASSWORD", true, logger)
	assert.NotNil(t, p.client)
	assert.NoError(t, err)

	_, err = NewPorkbunProvider(&domainFilter, "", "PASSWORD", true, logger)
	assert.Error(t, err)

	_, err = NewPorkbunProvider(&domainFilter, "KEY", "", true, logger)
	assert.Error(t, err)

	emptyDomainFilter := []string{}
	_, err = NewPorkbunProvider(&emptyDomainFilter, "KEY", "PASSWORD", true, logger)
	assert.Error(t, err)

}

func testApplyChanges(t *testing.T) {
	domainFilter := []string{"example.com"}
	var logger *slog.Logger
	promslogConfig := &promslog.Config{}
	logger = promslog.New(promslogConfig)

	p, _ := NewPorkbunProvider(&domainFilter, "KEY", "PASSWORD", true, logger)
	changes1 := &plan.Changes{
		Create:    []*endpoint.Endpoint{},
		Delete:    []*endpoint.Endpoint{},
		UpdateNew: []*endpoint.Endpoint{},
		UpdateOld: []*endpoint.Endpoint{},
	}

	// No Changes
	err := p.ApplyChanges(context.TODO(), changes1)
	assert.NoError(t, err)

	// Changes
	changes2 := &plan.Changes{
		Create: []*endpoint.Endpoint{
			{
				DNSName:    "api.example.com",
				RecordType: "A",
			},
			{
				DNSName:    "api.baz.com",
				RecordType: "TXT",
			}},
		Delete: []*endpoint.Endpoint{
			{
				DNSName:    "api.example.com",
				RecordType: "A",
			},
			{
				DNSName:    "api.baz.com",
				RecordType: "TXT",
			}},
		UpdateNew: []*endpoint.Endpoint{
			{
				DNSName:    "api.example.com",
				RecordType: "A",
			},
			{
				DNSName:    "api.baz.com",
				RecordType: "TXT",
			}},
		UpdateOld: []*endpoint.Endpoint{
			{
				DNSName:    "api.example.com",
				RecordType: "A",
			},
			{
				DNSName:    "api.baz.com",
				RecordType: "TXT",
			}},
	}

	err = p.ApplyChanges(context.TODO(), changes2)
	assert.NoError(t, err)
}

func testRecords(t *testing.T) {
	domainFilter := []string{"example.com"}
	var logger *slog.Logger
	promslogConfig := &promslog.Config{}
	logger = promslog.New(promslogConfig)

	p, _ := NewPorkbunProvider(&domainFilter, "KEY", "PASSWORD", true, logger)
	ep, err := p.Records(context.TODO())
	assert.Equal(t, []*endpoint.Endpoint{}, ep)
	assert.NoError(t, err)
}
