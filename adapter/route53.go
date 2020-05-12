package adapter

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/flood4life/dnser"
	"github.com/flood4life/dnser/config"
)

type Route53 struct {
	HostedZone string
	client     *route53.Route53
}

func NewRoute53(id, secret, hostedZone string) Route53 {
	s := session.Must(session.NewSession(
		aws.NewConfig().WithCredentials(
			credentials.NewStaticCredentials(id, secret, ""),
		),
	))
	svc := route53.New(s)
	return Route53{
		HostedZone: hostedZone,
		client:     svc,
	}
}

func (a Route53) List(ctx context.Context) ([]dnser.DNSRecord, error) {
	records := make([]dnser.DNSRecord, 0)
	err := a.client.ListResourceRecordSetsPagesWithContext(
		ctx,
		&route53.ListResourceRecordSetsInput{HostedZoneId: &a.HostedZone},
		func(output *route53.ListResourceRecordSetsOutput, _ bool) bool {
			for _, recordSet := range output.ResourceRecordSets {
				if *recordSet.Type != "A" {
					continue
				}
				if recordSet.AliasTarget != nil {
					records = append(records, dnser.DNSRecord{
						Alias:  true,
						Name:   config.Domain(*recordSet.Name),
						Target: config.Domain(*recordSet.AliasTarget.DNSName),
					})
				} else {
					for _, resourceRecord := range recordSet.ResourceRecords {
						records = append(records, dnser.DNSRecord{
							Alias:  false,
							Name:   config.Domain(*recordSet.Name),
							Target: config.Domain(*resourceRecord.Value),
						})
					}
				}
			}

			return true
		},
	)
	if err != nil {
		return nil, err
	}
	return records, nil
}
