// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/identity"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &AzdoProvider{}

// AzdoProvider defines the provider implementation.
type AzdoProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// AzdoProviderModel describes the provider data model.
type AzdoProviderModel struct {
	ServiceUrl          types.String `tfsdk:"org_service_url"`
	PersonalAccessToken types.String `tfsdk:"personal_access_token"`
}

func (p *AzdoProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "azdo"
	resp.Version = p.version
}

func (p *AzdoProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"org_service_url": schema.StringAttribute{
				Description: "The url of the Azure DevOps instance which should be used.",
				Optional:    true,
			},
			"personal_access_token": schema.StringAttribute{
				Description: "The personal access token which should be used",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *AzdoProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data AzdoProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	var serviceUrl string = os.Getenv("AZDO_ORG_SERVICE_URL")
	var personalAccessToken string = os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN")

	if !data.ServiceUrl.IsNull() {
		serviceUrl = data.ServiceUrl.ValueString()
	}

	if !data.PersonalAccessToken.IsNull() {
		personalAccessToken = data.PersonalAccessToken.ValueString()
	}

	if serviceUrl == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("serviceUrl"),
			"Missing Serviceurl",
			"The provider cannot create the AZDO API client as there is a missing or empty value for the AZDO API serviceurl. "+
				"Set the serviceUrl value in the configuration or use the AZDO_ORG_SERVICE_URL environment variable. ",
		)
	}

	if personalAccessToken == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("personalAccessToken"),
			"Missing personalAccessToken",
			"The provider cannot create the AZDO api as there is a missing or empty value for the AZDO API personalAccessToken. "+
				"Set the personalAccessToken value in the configuration or use the AZDO_PERSONAL_ACCESS_TOKEN environment variable.",
		)
	}

	log.Println("Service URL: ", serviceUrl)
	connection := azuredevops.NewPatConnection(serviceUrl, personalAccessToken)

	ctx = context.Background()

	// Create a client to interact with the Core area
	coreClient, err := identity.NewClient(ctx, connection)

	if err != nil {
		log.Fatal(err)
	}

	resp.DataSourceData = coreClient
	resp.ResourceData = coreClient

}

func (p *AzdoProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewGroupMembershipResource,
	}
}

func (p *AzdoProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewIdentitiesDataSource,
		NewIdentityDataSource,
	}
}

func (p *AzdoProvider) Functions(ctx context.Context) []func() function.Function {
	return nil
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AzdoProvider{
			version: version,
		}
	}
}
