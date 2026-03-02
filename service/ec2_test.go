package service_test

import (
    "fmt"
    "testing"

    "github.com/levelshatter/awsum/internal/app"
    "github.com/levelshatter/awsum/service"
    "github.com/stretchr/testify/assert"
)

func TestEC2_GetRunningInstances(t *testing.T) {
    app.Setup()

    instances, err := service.DefaultEC2.GetAllRunningInstances(t.Context())

    assert.NoError(t, err)

    for _, instance := range instances {
        fmt.Printf("%+v\n", instance.Info)
    }
}

func TestEC2_GetVPCs(t *testing.T) {
    app.Setup()

    vpcs, err := service.DefaultEC2.GetAllVPCs(t.Context())

    assert.NoError(t, err)
    assert.NotNil(t, vpcs)

    for _, vpc := range vpcs {
        fmt.Printf("%+v\n", vpc)
    }
}

func TestEC2_GetSubnets(t *testing.T) {
    app.Setup()

    subnets, err := service.DefaultEC2.GetAllSubnets(t.Context())

    assert.NoError(t, err)

    for _, subnet := range subnets {
        fmt.Printf("%+v\n", subnet)
    }
}
