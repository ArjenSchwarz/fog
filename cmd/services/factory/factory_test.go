package factory

import (
	"reflect"
	"testing"

	"github.com/ArjenSchwarz/fog/cmd/services/deployment"
	"github.com/ArjenSchwarz/fog/config"
)

// TestServiceFactory_CreateDeploymentService verifies dependency injection and
// accessor behaviour for the service factory.
func TestServiceFactory_CreateDeploymentService(t *testing.T) {
	cfg := &config.Config{}
	awsCfg := &config.AWSConfig{}
	f := NewServiceFactory(cfg, awsCfg)

	svc := f.CreateDeploymentService()
	ds, ok := svc.(*deployment.Service)
	if !ok {
		t.Fatalf("expected *deployment.Service, got %T", svc)
	}
	val := reflect.ValueOf(ds).Elem()
	if val.FieldByName("config").Pointer() != reflect.ValueOf(cfg).Pointer() {
		t.Errorf("config not injected")
	}
	if f.AppConfig() != cfg || f.AWSConfig() != awsCfg {
		t.Errorf("accessors returned wrong config")
	}
	if f.CreateDriftService() != nil || f.CreateStackService() != nil {
		t.Errorf("expected nil implementations")
	}
}

// TestServiceFactory_CreateDeploymentServiceNilConfig ensures a panic occurs
// when the AWS config is nil.
func TestServiceFactory_CreateDeploymentServiceNilConfig(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic")
		}
	}()
	f := NewServiceFactory(&config.Config{}, nil)
	f.CreateDeploymentService()
}
