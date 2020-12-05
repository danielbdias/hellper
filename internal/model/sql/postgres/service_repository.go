package postgres

import (
	"context"
	"errors"
	"fmt"
	"hellper/internal/log"
	"hellper/internal/model"
	"hellper/internal/model/sql"

	_ "github.com/lib/pq"
)

type serviceRepository struct {
	logger log.Logger
	db     sql.DB
}

// NewServiceRepository creates a new instance of a repository to handle services information
func NewServiceRepository(logger log.Logger, db sql.DB) model.ServiceRepository {
	return &serviceRepository{
		logger,
		db,
	}
}

// ListServiceInstances returns all service instances registered in the database
func (r *serviceRepository) ListServiceInstances(ctx context.Context) ([]*model.ServiceInstance, error) {
	query := `
	SELECT
		service_instance.id as id,
		(service.name || ' / ' || service_instance.name) as name
	FROM public.service
	INNER JOIN public.service_instance on service_instance.service_id = service.id
	`

	rows, err := r.db.Query(query)

	if err != nil {
		r.logger.Error(
			ctx,
			"postgres/service-repository.ListServiceInstances Query ERROR",
			log.NewValue("Error", err),
		)
		return nil, err
	}

	defer rows.Close()

	serviceInstances := make([]*model.ServiceInstance, 0)
	for rows.Next() {
		instance := model.ServiceInstance{}
		rows.Scan(&instance.ID, &instance.Name)
		serviceInstances = append(serviceInstances, &instance)
	}

	return serviceInstances, nil
}

// GetServiceInstance returns a specific service instance
func (r *serviceRepository) GetServiceInstance(ctx context.Context, instanceID int64) (*model.ServiceInstance, error) {
	query := `
	SELECT
    id,
    name
	FROM public.service_instance
  WHERE service_instance.id = $1
	`

	rows, err := r.db.Query(query, instanceID)
	if err != nil {
		r.logger.Error(
			ctx,
			"postgres/service-repository.GetServiceInstance Query ERROR",
			log.NewValue("instanceID", instanceID),
			log.NewValue("Error", err),
		)
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		err = errors.New("Service instance #" + fmt.Sprintf("%d", instanceID) + " not found")
		r.logger.Error(
			ctx,
			"postgres/incident-repository.GetServiceInstance ERROR",
			log.NewValue("instanceID", instanceID),
			log.NewValue("error", err),
		)
		return nil, err
	}

	var serviceInstance model.ServiceInstance
	rows.Scan(&serviceInstance.ID, &serviceInstance.Name)
	r.logger.Info(
		ctx,
		"postgres/incident-repository.GetServiceInstance SUCCESS",
		log.NewValue("instanceID", serviceInstance.ID),
		log.NewValue("instanceName", serviceInstance.Name),
	)

	return &serviceInstance, nil
}

// GetServiceInstanceOwner returns the owner team name of a service instance registered in the database
func (r *serviceRepository) GetServiceInstanceOwnerTeamName(ctx context.Context, instanceID int64) (string, error) {
	query := `
	SELECT
    team.name as team_name
	FROM public.service_instance
	INNER JOIN public.team on service_instance.owner_team_id = team.id
  WHERE service_instance.id = $1
	`

	rows, err := r.db.Query(query, instanceID)
	if err != nil {
		r.logger.Error(
			ctx,
			"postgres/service-repository.GetServiceInstanceOwnerTeamName Query ERROR",
			log.NewValue("instanceID", instanceID),
			log.NewValue("Error", err),
		)
		return "", err
	}
	defer rows.Close()

	if !rows.Next() {
		err = errors.New("Owner team of service instance #" + fmt.Sprintf("%d", instanceID) + " not found")
		r.logger.Error(
			ctx,
			"postgres/incident-repository.GetServiceInstanceOwnerTeamName ERROR",
			log.NewValue("instanceID", instanceID),
			log.NewValue("error", err),
		)
		return "", err
	}

	var ownerTeamName string
	rows.Scan(&ownerTeamName)
	r.logger.Info(
		ctx,
		"postgres/incident-repository.GetServiceInstanceOwnerTeamName SUCCESS",
		log.NewValue("instanceID", instanceID),
		log.NewValue("ownerTeamName", ownerTeamName),
	)

	return ownerTeamName, nil
}
