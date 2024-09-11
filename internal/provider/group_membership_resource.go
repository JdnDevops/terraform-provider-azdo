// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"slices"
	"terraform-provider-azdo/services"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/microsoft/azure-devops-go-api/azuredevops/identity"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &GroupMembershipResource{}
var _ resource.ResourceWithImportState = &GroupMembershipResource{}

func NewGroupMembershipResource() resource.Resource {
	return &GroupMembershipResource{}
}

// GroupMembershipResource defines the resource implementation.
type GroupMembershipResource struct {
	client *identity.ClientImpl
}

// GroupMembershipResourceModel describes the resource data model.
type GroupMembershipResourceModel struct {
	ProjectId types.String   `tfsdk:"project_id"`
	Group     types.String   `tfsdk:"group"`
	Members   []types.String `tfsdk:"members"`
}

func (r *GroupMembershipResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_membership"
}

func (r *GroupMembershipResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Azdo Groupmembership resource",

		Attributes: map[string]schema.Attribute{
			"group": schema.StringAttribute{
				MarkdownDescription: "Group to manage membership for",
				Optional:            false,
				Required:            true,
			},
			"members": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of members to add to the group",
				Optional:            false,
				Required:            true,
				Computed:            false,
			},
			"project_id": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				MarkdownDescription: "Unique identifier for the group membership",
			},
		},
	}
}

func (r *GroupMembershipResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*identity.ClientImpl)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *identity.ClientImpl, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *GroupMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GroupMembershipResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	identityService := services.NewIdentityService(r.client)

	var foundmembers []*identity.Identity
	for _, member := range data.Members {
		var response, error = identityService.GetIdentityByName(ctx, member.ValueString())

		if error != nil {
			resp.Diagnostics.AddError("error", error.Error())
			return
		}
		foundmembers = append(foundmembers, response)
	}

	var foundGroup, err = identityService.GetGroup(ctx, data.Group.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error", err.Error())
		return
	}

	for _, foundIdentity := range foundmembers {
		err := identityService.AddMemberToGroup(ctx, foundGroup, foundIdentity)
		if err != nil {
			resp.Diagnostics.AddError("Error", err.Error())
			return
		}
		tflog.Info(ctx, fmt.Sprintf("Added member %s to group: %s", *foundIdentity.ProviderDisplayName, *foundGroup.ProviderDisplayName))
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GroupMembershipResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// it seems reading from the data source is not necessary. If I read from the data source, it somehow always updates the state. even if nothing has changed

	// identityService := services.NewIdentityService(r.client)

	// var members, error = identityService.GetGroupMembers(ctx, data.Group.ValueString())
	// if error != nil {
	// 	resp.Diagnostics.AddError("Error", error.Error())
	// 	return
	// }

	// var currentMembers []types.String
	// for _, member := range *members {
	// 	for _, stateMember := range data.Members {
	// 		if stateMember.ValueString() == *member.CustomDisplayName {
	// 			currentMembers = append(currentMembers, stateMember)
	// 		}
	// 	}
	// }

	// data.Members = currentMembers

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data GroupMembershipResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	identityService := services.NewIdentityService(r.client)

	var members, error = identityService.GetGroupMembers(ctx, data.Group.ValueString())
	if error != nil {
		resp.Diagnostics.AddError("Error", error.Error())
		return
	}

	var foundGroup, err = identityService.GetGroup(ctx, data.Group.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error", err.Error())
		return
	}

	var toAddMembers []identity.Identity
	for _, stateMember := range data.Members {
		containsMember := slices.ContainsFunc(*members, func(m identity.Identity) bool {
			return *m.CustomDisplayName == stateMember.ValueString()
		})

		if !containsMember {
			foundMember, err := identityService.GetIdentityByName(ctx, stateMember.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Error", err.Error())
				return
			}
			toAddMembers = append(toAddMembers, *foundMember)
		}
	}

	var toRemoveMembers []identity.Identity
	for _, member := range *members {

		containsMember := slices.Contains(data.Members, types.StringValue(*member.CustomDisplayName))
		if !containsMember {
			toRemoveMembers = append(toRemoveMembers, member)
		}
	}

	for _, foundIdentity := range toAddMembers {
		err := identityService.AddMemberToGroup(ctx, foundGroup, &foundIdentity)
		if err != nil {
			resp.Diagnostics.AddError("Error", err.Error())
			return
		}
		tflog.Info(ctx, fmt.Sprintf("added member %s to group: %s", *foundIdentity.CustomDisplayName, *foundIdentity.CustomDisplayName))
	}

	for _, foundIdentity := range toRemoveMembers {
		err := identityService.RemoveMemberFromGroup(ctx, foundGroup, &foundIdentity)
		if err != nil {
			resp.Diagnostics.AddError("Error", err.Error())
			return
		}
		tflog.Info(ctx, fmt.Sprintf("removed member %s from group: %s", *foundIdentity.CustomDisplayName, *foundIdentity.CustomDisplayName))
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GroupMembershipResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete example, got error: %s", err))
	//     return
	// }
}

func (r *GroupMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
