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
	err := a.client.ListResourceRecordSetsPagesWithContext(ctx, a.listInput(),
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

func (a Route53) listInput() *route53.ListResourceRecordSetsInput {
	return &route53.ListResourceRecordSetsInput{
		HostedZoneId: &a.HostedZone,
	}
}

func (a Route53) Process(ctx context.Context, actions dnser.Actions) error {
	_, err := a.client.ChangeResourceRecordSetsWithContext(ctx, a.changeSetInput(actions))
	return err
}

func (a Route53) changeSetInput(actions dnser.Actions) *route53.ChangeResourceRecordSetsInput {
	return &route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  a.changeBatch(actions),
		HostedZoneId: &a.HostedZone,
	}
}

func (a Route53) changeBatch(actions dnser.Actions) *route53.ChangeBatch {
	return &route53.ChangeBatch{
		Changes: append(
			a.upsertActions(actions.PutActions),
			a.deleteActions(actions.DeleteActions)...,
		),
	}
}

func (a Route53) changeActions(kind string, actions []dnser.DNSRecord) []*route53.Change {
	result := make([]*route53.Change, len(actions))

	for i, action := range actions {
		result[i] = &route53.Change{
			Action:            aws.String(kind),
			ResourceRecordSet: a.resourceRecordSet(action),
		}
	}

	return result
}

func (a Route53) upsertActions(actions []dnser.DNSRecord) []*route53.Change {
	return a.changeActions("UPSERT", actions)
}

func (a Route53) deleteActions(actions []dnser.DNSRecord) []*route53.Change {
	return a.changeActions("DELETE", actions)
}

func (a Route53) resourceRecordSet(record dnser.DNSRecord) *route53.ResourceRecordSet {
	if record.Alias {
		return a.aliasRecord(record)
	}
	return &route53.ResourceRecordSet{
		Name:            recordName(record),
		ResourceRecords: []*route53.ResourceRecord{{Value: recordTarget(record)}},
		Type:            typeA(),
	}
}

func (a Route53) aliasRecord(record dnser.DNSRecord) *route53.ResourceRecordSet {
	return &route53.ResourceRecordSet{
		AliasTarget: &route53.AliasTarget{
			DNSName:              recordTarget(record),
			EvaluateTargetHealth: aws.Bool(false),
			HostedZoneId:         aws.String(a.HostedZone),
		},
		Name: recordName(record),
		Type: typeA(),
	}
}

func recordName(r dnser.DNSRecord) *string {
	return aws.String(string(r.Name))
}

func recordTarget(r dnser.DNSRecord) *string {
	return aws.String(string(r.Target))
}

func typeA() *string {
	return aws.String("A")
}
