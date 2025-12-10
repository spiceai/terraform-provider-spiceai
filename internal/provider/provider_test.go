// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"spiceai": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// Check that required environment variables are set for acceptance tests
	if v := os.Getenv("SPICEAI_CLIENT_ID"); v == "" {
		t.Fatal("SPICEAI_CLIENT_ID must be set for acceptance tests")
	}
	if v := os.Getenv("SPICEAI_CLIENT_SECRET"); v == "" {
		t.Fatal("SPICEAI_CLIENT_SECRET must be set for acceptance tests")
	}
}

func TestProviderNew(t *testing.T) {
	p := New("test")()

	if p == nil {
		t.Fatal("expected provider to be non-nil")
	}

	sp, ok := p.(*SpiceAIProvider)
	if !ok {
		t.Fatalf("expected provider to be *SpiceAIProvider, got %T", p)
	}

	if sp.version != "test" {
		t.Errorf("expected version to be 'test', got '%s'", sp.version)
	}
}
