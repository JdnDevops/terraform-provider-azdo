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

func NewIdentitiesDataSource() datasource.DataSource {
	log.Println("NewIdentitiesDataSource")
	return &IdentitiesDataSource{}
}

// IdentitiesDataSource defines the data source implementation.
type IdentitiesDataSource struct {
	client *identity.ClientImpl
}

// IdentitiesDataSourceModel describes the data source data model.
type IdentitiesDataSourceModel struct {
	Identities []IdentityModel `tfsdk:"identities"`
}

type IdentityModel struct {
	Id                types.String `tfsdk:"id"`
	DisplayName       types.String `tfsdk:"display_name"`
	SubjectDescriptor types.String `tfsdk:"subject_descriptor"`
	Descriptor        types.String `tfsdk:"descriptor"`
}

func (d *IdentitiesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identities"
}

func (d *IdentitiesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Azdo Identity",
		Attributes: map[string]schema.Attribute{
			"identities": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The identity ID",
						},
						"display_name": schema.StringAttribute{
							Computed:    true,
							Description: "The display name of the identity",
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
				},
			},
		},
	}
}

func (d *IdentitiesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	log.Println("Configure IdentitiesDataSource")
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

func (d *IdentitiesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IdentitiesDataSourceModel = IdentitiesDataSourceModel{}

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	var response, error = d.client.ListGroups(ctx, identity.ListGroupsArgs{})
	if error != nil {
		resp.Diagnostics.AddError("Error", error.Error())
		return
	}
	for _, group := range *response {
		var groupId = group.Id.String()
		var identity, err = d.client.ReadIdentity(ctx, identity.ReadIdentityArgs{IdentityId: &groupId})
		if err != nil {
			resp.Diagnostics.AddError("Error", err.Error())
			continue
		}
		identityModel := IdentityModel{
			Id: types.StringValue(identity.Id.String()),
		}
		if identity.ProviderDisplayName != nil {
			identityModel.DisplayName = types.StringValue(*identity.ProviderDisplayName)
		}
		if identity.SubjectDescriptor != nil {
			identityModel.SubjectDescriptor = types.StringValue(*identity.SubjectDescriptor)
		}
		if identity.Descriptor != nil {
			identityModel.Descriptor = types.StringValue(*identity.Descriptor)
		}
		data.Identities = append(data.Identities, identityModel)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
