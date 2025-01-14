// resource_ziti_service_test.go

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/openziti/sdk-golang/edge-apis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockClient struct {
	mock.Mock
}

func (m *MockClient) CreateService(params *service.NewCreateServiceParams, _ ...interface{}) (*service.CreateServiceOK, error) {
	args := m.Called(params)
	return args.Get(0).(*service.CreateServiceOK), args.Error(1)
}

func (m *MockClient) DetailService(params *service.NewDetailServiceParams, _ ...interface{}) (*service.DetailServiceOK, error) {
	args := m.Called(params)
	return args.Get(0).(*service.DetailServiceOK), args.Error(1)
}

func (m *MockClient) UpdateService(params *service.NewUpdateServiceParams, _ ...interface{}) (*service.UpdateServiceOK, error) {
	args := m.Called(params)
	return args.Get(0).(*service.UpdateServiceOK), args.Error(1)
}

func (m *MockClient) DeleteService(params *service.NewDeleteServiceParams, _ ...interface{}) (*service.DeleteServiceOK, error) {
	args := m.Called(params)
	return args.Get(0).(*service.DeleteServiceOK), args.Error(1)
}

func TestZitiServiceResource_Create(t *testing.T) {
	mockClient := new(MockClient)
	resource := &ZitiServiceResource{
		client: mockClient,
	}

	expectedID := "service-id"
	mockClient.On("CreateService", mock.Anything).Return(&service.CreateServiceOK{
		Payload: &service.CreateResponse{
			Data: &rest_model.Service{
				ID: &expectedID,
			},
		},
	}, nil)

	// Define the terraform plan input
	plan := ZitiServiceResourceModel{
		Name:                   types.StringValue("test-service"),
		Configs:                types.ListValueMust(types.ListType, []types.String{types.StringValue("config-id")}),
		EncryptionRequired:     types.BoolValue(true),
		MaxIdleTimeMilliseconds: types.Int64Value(30000),
		RoleAttributes:         types.ListValueMust(types.ListType, []types.String{}),
		TerminatorStrategy:     types.StringValue("smartrouting"),
	}

	// Invoke the Create method on the resource
	resp := resource.Create(context.Background(), resource.CreateRequest{
		Plan: plan,
	})

	// Assertions
	assert.NoError(t, resp.Diagnostics)
	assert.Equal(t, expectedID, plan.ID.ValueString())

	// Verify the mock call
	mockClient.AssertExpectations(t)
}

func TestZitiServiceResource_Read(t *testing.T) {
	mockClient := new(MockClient)
	resource := &ZitiServiceResource{
		client: mockClient,
	}

	serviceDetail := &rest_model.Service{
		ID:               &expectedID,
		Name:             "test-service",
		Configs:          []string{"config-id"},
		EncryptionRequired: true,
		MaxIdleTimeMillis: 30000,
		// Add other fields as required.
	}

	mockClient.On("DetailService", mock.Anything).Return(&service.DetailServiceOK{
		Payload: &service.DetailServiceResponse{
			Data: serviceDetail,
		},
	}, nil)

	// Define the terraform state input
	state := ZitiServiceResourceModel{
		ID: types.StringValue(expectedID), // The ID must be set for Read to know which resource to query.
	}

	// Invoke the Read method on the resource
	resp := resource.Read(context.Background(), resource.ReadRequest{
		State: state,
	})

	// Assertions
	assert.NoError(t, resp.Diagnostics)
	assert.Equal(t, "test-service", state.Name.ValueString())
	assert.Equal(t, 30000, state.MaxIdleTimeMilliseconds.ValueInt64())

	// Verify the mock call
	mockClient.AssertExpectations(t)
}

// Tests for Update and Delete would follow a similar pattern.
// You would mock appropriate methods in MockClient, set expectations, invoke the methods on the resource, and assert results.
