package adapter

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/flood4life/dnser"
	"github.com/flood4life/dnser/config"
	"golang.org/x/sync/errgroup"
)

const defaultTTL = 300 // 5 minutes

// Route53 is an Adapter that's using AWS Route53.
type Route53 struct {
	client *route53.Client
	zones  map[config.Domain]string // map TLDs to zone IDs
}

type hostedZone struct {
	id   string
	name config.Domain
}

// NewRoute53 constructs a Route53 instance from AWS access key and secret.
func NewRoute53(id, secret string) Route53 {
	cfg, err := awsConfig.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(err)
	}
	creds := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
		id, secret, "",
	))
	cfg.Credentials = creds
	cfg.Region = "eu-west-1"
	return NewRoute53FromSession(cfg)
}

// NewRoute53FromSession could be used if a more detailed configuration of AWS Session is needed.
func NewRoute53FromSession(c aws.Config) Route53 {
	return Route53{
		client: route53.NewFromConfig(c),
		zones:  make(map[config.Domain]string),
	}
}

// List returns all DNS records from all Hosted Zones.
func (a Route53) List(ctx context.Context) ([]dnser.DNSRecord, error) {
	records, err := a.recordsPerZone(ctx)
	if err != nil {
		return nil, err
	}

	return flattenRecords(records), nil
}

func (a Route53) zoneIDFromDomain(domain config.Domain) string {
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
	paginator := route53.NewListHostedZonesPaginator(a.client, listZonesInput())
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, zone := range output.HostedZones {
			zones = append(zones, hostedZone{
				id:   extractZoneID(*zone.Id),
				name: config.Domain(*zone.Name),
			})
		}
	}
	return zones, nil
}

func (a Route53) listZoneRecords(ctx context.Context, zone hostedZone) ([]dnser.DNSRecord, error) {
	records := make([]dnser.DNSRecord, 0)
	paginator := NewListResourceRecordSetsPaginator(a.client, listZoneRecordsInput(zone.id))
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, recordSet := range output.ResourceRecordSets {
			if recordSet.Type != types.RRTypeA {
				continue
			}
			if recordSet.AliasTarget != nil {
				records = append(records, dnser.NewAliasRecord(
					*recordSet.Name, *recordSet.AliasTarget.DNSName,
				))
				continue
			}
			for _, resourceRecord := range recordSet.ResourceRecords {
				records = append(records, dnser.NewRecord(
					*recordSet.Name, *resourceRecord.Value,
				))
			}
		}
	}
	return records, nil
}

func listZonesInput() *route53.ListHostedZonesInput {
	return &route53.ListHostedZonesInput{}
}

func listZoneRecordsInput(zoneID string) *route53.ListResourceRecordSetsInput {
	return &route53.ListResourceRecordSetsInput{
		HostedZoneId: &zoneID,
	}
}

// Process creates and deletes DNS records.
func (a Route53) Process(ctx context.Context, actionGroups [][]dnser.Action) error {
	for _, actions := range actionGroups {
		g, gCtx := errgroup.WithContext(ctx)
		inputs := a.changeSetInputs(actions)
		for _, input := range inputs {
			a.processChangeSet(gCtx, g, input)
		}

		if err := g.Wait(); err != nil {
			return err
		}
	}

	return nil
}

func (a Route53) processChangeSet(ctx context.Context, g *errgroup.Group, input *route53.ChangeResourceRecordSetsInput) {
	g.Go(func() error {
		res, err := a.client.ChangeResourceRecordSets(ctx, input)
		if err != nil {
			return err
		}
		for {
			// wait for changes to become active
			change, err := a.client.GetChange(ctx, &route53.GetChangeInput{Id: res.ChangeInfo.Id})
			if err != nil {
				return err
			}
			if change.ChangeInfo.Status == types.ChangeStatusInsync {
				break
			}
			time.Sleep(time.Second)
		}
		return nil
	})
}

func (a Route53) changeSetInputs(actions []dnser.Action) []*route53.ChangeResourceRecordSetsInput {
	result := make([]*route53.ChangeResourceRecordSetsInput, 0)

	groupedActions := a.groupActionsPerTLD(actions)
	for zoneID, zoneActions := range groupedActions {
		result = append(result, &route53.ChangeResourceRecordSetsInput{
			ChangeBatch:  a.changeBatch(zoneActions),
			HostedZoneId: &zoneID,
		})
	}

	return result
}

func (a Route53) groupActionsPerTLD(actions []dnser.Action) map[string][]dnser.Action {
	result := make(map[string][]dnser.Action)
	for _, action := range actions {
		zoneID := a.zoneIDFromDomain(action.Record.NameZone())
		if _, ok := result[zoneID]; !ok {
			result[zoneID] = make([]dnser.Action, 0)
		}
		result[zoneID] = append(result[zoneID], action)
	}
	return result
}

func (a Route53) changeBatch(actions []dnser.Action) *types.ChangeBatch {
	return &types.ChangeBatch{
		Changes: a.changeActions(actions),
	}
}

func (a Route53) changeActions(actions []dnser.Action) []types.Change {
	result := make([]types.Change, len(actions))

	for i, action := range actions {
		result[i] = types.Change{
			Action:            actionFromActionType(action.Type),
			ResourceRecordSet: a.resourceRecordSet(action.Record),
		}
	}

	return result
}

func actionFromActionType(actionType dnser.ActionType) types.ChangeAction {
	switch actionType {
	case dnser.Delete:
		return types.ChangeActionDelete
	case dnser.Upsert:
		return types.ChangeActionUpsert
	default:
		panic("don't know how to handle dnser action type: " + actionType)
	}
}

func (a Route53) resourceRecordSet(record dnser.DNSRecord) *types.ResourceRecordSet {
	if record.Alias {
		return a.aliasRecord(record)
	}
	return &types.ResourceRecordSet{
		Name:            recordName(record),
		ResourceRecords: []types.ResourceRecord{{Value: recordTarget(record)}},
		TTL:             aws.Int64(defaultTTL),
		Type:            types.RRTypeA,
	}
}

func (a Route53) aliasRecord(record dnser.DNSRecord) *types.ResourceRecordSet {
	return &types.ResourceRecordSet{
		AliasTarget: &types.AliasTarget{
			DNSName:              recordTarget(record),
			EvaluateTargetHealth: true,
			HostedZoneId:         aws.String(a.zoneIDFromDomain(record.NameZone())),
		},
		Name: recordName(record),
		Type: types.RRTypeA,
	}
}

func recordName(r dnser.DNSRecord) *string {
	return aws.String(string(r.Name))
}

func recordTarget(r dnser.DNSRecord) *string {
	return aws.String(string(r.Target))
}

func extractZoneID(response string) string {
	// zone ID from aws response looks like "/hostedzone/Z2H9GA7I9LH893",
	// but it expects the input in "Z2H9GA7I9LH893" format
	lastSlashIndex := strings.LastIndex(response, "/")
	return response[lastSlashIndex+1:]
}
