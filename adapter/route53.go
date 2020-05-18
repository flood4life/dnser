package adapter

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/flood4life/dnser"
	"github.com/flood4life/dnser/config"
	"golang.org/x/sync/errgroup"
)

const typeA = "A"
const defaultTTL = 300 // 5 minutes

type Route53 struct {
	client *route53.Route53
	zones  map[config.Domain]string // map TLDs to zone IDs
}

type hostedZone struct {
	id   string
	name config.Domain
}

func NewRoute53(id, secret string) Route53 {
	s := session.Must(session.NewSession(
		aws.NewConfig().WithCredentials(
			credentials.NewStaticCredentials(id, secret, ""),
		),
	))
	return NewRoute53FromSession(s)
}

func NewRoute53FromSession(s *session.Session) Route53 {
	return Route53{
		client: route53.New(s),
		zones:  make(map[config.Domain]string),
	}
}

func (a Route53) List(ctx context.Context) ([]dnser.DNSRecord, error) {
	records, err := a.recordsPerZone(ctx)
	if err != nil {
		return nil, err
	}

	return flattenRecords(records), nil
}

func (a Route53) zoneIdForTLD(domain config.Domain) string {
	// TODO: handle cases when map is not initialized
	id, ok := a.zones[domain]
	if !ok {
		panic("zone ID is unknown")
	}
	return id
}

func (a Route53) initZonesMap(zones []hostedZone) {
	for _, zone := range zones {
		a.zones[zone.name] = zone.id
	}
}

func flattenRecords(records [][]dnser.DNSRecord) []dnser.DNSRecord {
	result := make([]dnser.DNSRecord, 0)
	for _, zoneRecords := range records {
		result = append(result, zoneRecords...)
	}
	return result
}

func (a Route53) recordsPerZone(ctx context.Context) ([][]dnser.DNSRecord, error) {
	zones, err := a.listHostedZones(ctx)
	if err != nil {
		return nil, err
	}
	a.initZonesMap(zones)

	g, ctx := errgroup.WithContext(ctx)
	records := make([][]dnser.DNSRecord, len(zones))

	for i, zone := range zones {
		i, zone := i, zone
		g.Go(func() error {
			zoneRecords, err := a.listZoneRecords(ctx, zone)
			if err == nil {
				records[i] = zoneRecords
			}
			return err
		})
	}

	err = g.Wait()
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (a Route53) listHostedZones(ctx context.Context) ([]hostedZone, error) {
	zones := make([]hostedZone, 0)
	err := a.client.ListHostedZonesPagesWithContext(ctx, listZonesInput(),
		func(output *route53.ListHostedZonesOutput, _ bool) bool {
			for _, zone := range output.HostedZones {
				zones = append(zones, hostedZone{
					id:   extractZoneId(*zone.Id),
					name: config.Domain(*zone.Name),
				})
			}

			return true
		})
	return zones, err
}

func extractZoneId(response string) string {
	// zone ID from aws response looks like "/hostedzone/Z2H9GA7I9LH893",
	// but it expects the input in "Z2H9GA7I9LH893" format
	lastSlashIndex := strings.LastIndex(response, "/")
	return response[lastSlashIndex+1:]
}

func (a Route53) listZoneRecords(ctx context.Context, zone hostedZone) ([]dnser.DNSRecord, error) {
	records := make([]dnser.DNSRecord, 0)
	err := a.client.ListResourceRecordSetsPagesWithContext(ctx, listZoneRecordsInput(zone.id),
		func(output *route53.ListResourceRecordSetsOutput, _ bool) bool {
			for _, recordSet := range output.ResourceRecordSets {
				if *recordSet.Type != typeA {
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

func listZonesInput() *route53.ListHostedZonesInput {
	return &route53.ListHostedZonesInput{}
}

func listZoneRecordsInput(zoneId string) *route53.ListResourceRecordSetsInput {
	return &route53.ListResourceRecordSetsInput{
		HostedZoneId: &zoneId,
	}
}

func (a Route53) Process(ctx context.Context, actions []dnser.Action) error {
	inputs := a.changeSetInputs(actions)
	g, ctx := errgroup.WithContext(ctx)
	for _, input := range inputs {
		input := input
		g.Go(func() error {
			_, err := a.client.ChangeResourceRecordSetsWithContext(ctx, input)
			return err
		})
	}
	return g.Wait()
}

func (a Route53) changeSetInputs(actions []dnser.Action) []*route53.ChangeResourceRecordSetsInput {
	result := make([]*route53.ChangeResourceRecordSetsInput, 0)

	groupedActions := a.groupActionsPerTLD(actions)
	for zoneId, zoneActions := range groupedActions {
		result = append(result, &route53.ChangeResourceRecordSetsInput{
			ChangeBatch:  a.changeBatch(zoneActions),
			HostedZoneId: &zoneId,
		})
	}

	return result
}

func (a Route53) groupActionsPerTLD(actions []dnser.Action) map[string][]dnser.Action {
	result := make(map[string][]dnser.Action)
	for _, action := range actions {
		zoneId := a.zoneIdForTLD(action.Record.NameTLD())
		if _, ok := result[zoneId]; !ok {
			result[zoneId] = make([]dnser.Action, 0)
		}
		result[zoneId] = append(result[zoneId], action)
	}
	return result
}

func (a Route53) changeBatch(actions []dnser.Action) *route53.ChangeBatch {
	return &route53.ChangeBatch{
		Changes: a.changeActions(actions),
	}
}

func (a Route53) changeActions(actions []dnser.Action) []*route53.Change {
	result := make([]*route53.Change, len(actions))

	for i, action := range actions {
		result[i] = &route53.Change{
			Action:            aws.String(actionFromActionType(action.Type)),
			ResourceRecordSet: a.resourceRecordSet(action.Record),
		}
	}

	return result
}

func actionFromActionType(actionType dnser.ActionType) string {
	switch actionType {
	case dnser.Delete:
		return "DELETE"
	case dnser.Upsert:
		return "UPSERT"
	default:
		return ""
	}
}

func (a Route53) resourceRecordSet(record dnser.DNSRecord) *route53.ResourceRecordSet {
	if record.Alias {
		return a.aliasRecord(record)
	}
	return &route53.ResourceRecordSet{
		Name:            recordName(record),
		ResourceRecords: []*route53.ResourceRecord{{Value: recordTarget(record)}},
		TTL:             aws.Int64(defaultTTL),
		Type:            aws.String(typeA),
	}
}

func (a Route53) aliasRecord(record dnser.DNSRecord) *route53.ResourceRecordSet {
	return &route53.ResourceRecordSet{
		AliasTarget: &route53.AliasTarget{
			DNSName:              recordTarget(record),
			EvaluateTargetHealth: aws.Bool(false),
			HostedZoneId:         aws.String(a.zoneIdForTLD(record.NameTLD())),
		},
		Name: recordName(record),
		Type: aws.String(typeA),
	}
}

func recordName(r dnser.DNSRecord) *string {
	return aws.String(string(r.Name))
}

func recordTarget(r dnser.DNSRecord) *string {
	return aws.String(string(r.Target))
}
