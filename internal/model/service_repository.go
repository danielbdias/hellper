package model

import "context"

// ServiceRepository wraps services data from the database
type ServiceRepository interface {
	ListServiceInstances(ctx context.Context) ([]*ServiceInstance, error)
	GetServiceInstance(ctx context.Context, instanceID int64) (*ServiceInstance, error)
	GetServiceInstanceOwnerTeamName(ctx context.Context, instanceID int64) (string, error)
}
