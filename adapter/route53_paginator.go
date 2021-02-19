package adapter

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

// ListResourceRecordSetsAPIClient is a client that implements the ListResourceRecordSets
// operation.
type ListResourceRecordSetsAPIClient interface {
	ListResourceRecordSets(context.Context, *route53.ListResourceRecordSetsInput, ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error)
}

var _ ListResourceRecordSetsAPIClient = (*route53.Client)(nil)

// ListResourceRecordSetsPaginatorOptions is the paginator options for ListResourceRecordSets
type ListResourceRecordSetsPaginatorOptions struct {
	// (Optional) The maximum number of hosted zones that you want Amazon Route 53 to
	// return. If you have more than maxitems hosted zones, the value of IsTruncated in
	// the response is true, and the value of NextMarker is the hosted zone ID of the
	// first hosted zone that Route 53 will return if you submit another request.
	Limit int32
}

// ListResourceRecordSetsPaginator is a paginator for ListResourceRecordSets
type ListResourceRecordSetsPaginator struct {
	options ListResourceRecordSetsPaginatorOptions
	client  ListResourceRecordSetsAPIClient
	params  *route53.ListResourceRecordSetsInput

	nextRecordIdentifier *string
	nextRecordName       *string
	nextRecordType       types.RRType

	firstPage bool
}

// NewListResourceRecordSetsPaginator returns a new ListResourceRecordSetsPaginator
func NewListResourceRecordSetsPaginator(client ListResourceRecordSetsAPIClient, params *route53.ListResourceRecordSetsInput, optFns ...func(*ListResourceRecordSetsPaginatorOptions)) *ListResourceRecordSetsPaginator {
	options := ListResourceRecordSetsPaginatorOptions{}
	if params.MaxItems != nil {
		options.Limit = *params.MaxItems
	}

	for _, fn := range optFns {
		fn(&options)
	}

	if params == nil {
		params = &route53.ListResourceRecordSetsInput{}
	}

	return &ListResourceRecordSetsPaginator{
		options:   options,
		client:    client,
		params:    params,
		firstPage: true,
	}
}

// HasMorePages returns a boolean indicating whether more pages are available
func (p *ListResourceRecordSetsPaginator) HasMorePages() bool {
	return p.firstPage || p.nextRecordIdentifier != nil || p.nextRecordType != "" || p.nextRecordName != nil
}

// NextPage retrieves the next ListResourceRecordSets page.
func (p *ListResourceRecordSetsPaginator) NextPage(ctx context.Context, optFns ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error) {
	if !p.HasMorePages() {
		return nil, fmt.Errorf("no more pages available")
	}

	params := *p.params
	params.StartRecordIdentifier = p.nextRecordIdentifier
	params.StartRecordName = p.nextRecordName
	params.StartRecordType = p.nextRecordType

	var limit *int32
	if p.options.Limit > 0 {
		limit = &p.options.Limit
	}
	params.MaxItems = limit

	result, err := p.client.ListResourceRecordSets(ctx, &params, optFns...)
	if err != nil {
		return nil, err
	}
	p.firstPage = false

	p.nextRecordIdentifier = result.NextRecordIdentifier
	p.nextRecordName = result.NextRecordName
	p.nextRecordType = result.NextRecordType

	return result, nil
}
