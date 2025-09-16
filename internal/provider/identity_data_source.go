// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/microsoft/azure-devops-go-api/azuredevops/identity"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &IdentitiesDataSource{}

func NewIdentityDataSource() datasource.DataSource {
	log.Println("NewIdentityDataSource")
	return &IdentityDataSource{}
}

// IdentitiesDataSource defines the data source implementation.
type IdentityDataSource struct {
	client *identity.ClientImpl
}

func (d *IdentityDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity"
}

func (d *IdentityDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Azdo Identity",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The identity ID",
			},
			"display_name": schema.StringAttribute{
				Description: "The display name of the identity",
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The project ID (this is a bandaid fix to allow Terraform to use data sources on non-created projects)",
				Optional:    true,
			},
			"subject_descriptor": schema.StringAttribute{
				Computed:    true,
				Description: "The subject descriptor of the identity",
			},
			"descriptor": schema.StringAttribute{
				Computed:    true,
				Description: "The descriptor of the identity",
			},
		},
	}
}

func (d *IdentityDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	log.Println("Configure IdentityDataSource")
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*identity.ClientImpl)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *identity.ClientImpl, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *IdentityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IdentityModel = IdentityModel{}

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.DisplayName.IsUnknown() || data.DisplayName.IsNull() && data.DisplayName == types.StringValue("") {
		resp.Diagnostics.AddError("Error", "display_name is required")
		return
	}
	recurse := true
	var response, error = d.client.ListGroups(ctx, identity.ListGroupsArgs{Recurse: &recurse})
	if error != nil {
		resp.Diagnostics.AddError("Error", error.Error())
		return
	}

	var foundGroupId string

	for _, group := range *response {
		if group.ProviderDisplayName != nil && *group.ProviderDisplayName == data.DisplayName.ValueString() {
			foundGroupId = group.Id.String()
			break
		}
	}

	if foundGroupId == "" {
		resp.Diagnostics.AddError("Error", "Group "+data.DisplayName.ValueString()+" not found")
		return
	}

	var identity, err = d.client.ReadIdentity(ctx, identity.ReadIdentityArgs{IdentityId: &foundGroupId})
	if err != nil {
		resp.Diagnostics.AddError("Error", err.Error())
	}
	data = IdentityModel{
		Id: types.StringValue(identity.Id.String()),
	}
	if identity.ProviderDisplayName != nil {
		data.DisplayName = types.StringValue(*identity.ProviderDisplayName)
	}
	if identity.SubjectDescriptor != nil {
		data.SubjectDescriptor = types.StringValue(*identity.SubjectDescriptor)
	}
	if identity.Descriptor != nil {
		data.Descriptor = types.StringValue(*identity.Descriptor)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
