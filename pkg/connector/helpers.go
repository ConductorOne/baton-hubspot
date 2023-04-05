package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/pagination"
)

var ResourcesPageSize = 50

func parsePageToken(i string, resourceID *v2.ResourceId) (*pagination.Bag, error) {
	b := &pagination.Bag{}
	err := b.Unmarshal(i)
	if err != nil {
		return nil, err
	}

	if b.Current() == nil {
		b.Push(pagination.PageState{
			ResourceTypeID: resourceID.ResourceType,
			ResourceID:     resourceID.Resource,
		})
	}

	return b, nil
}

func filter[T any](slice []T, predicate func(value T) bool) []T {
	var result []T
	for _, value := range slice {
		if predicate(value) {
			result = append(result, value)
		}
	}

	return result
}

func includes(slice []string, target string) bool {
	for _, value := range slice {
		if value == target {
			return true
		}
	}

	return false
}
