package services

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/microsoft/azure-devops-go-api/azuredevops/identity"
)

func NewIdentityService(client *identity.ClientImpl) *IdentityService {
	return &IdentityService{client: client}
}

type IdentityService struct {
	client *identity.ClientImpl
}

func (s *IdentityService) GetIdentityByName(ctx context.Context, name string) (*identity.Identity, error) {
	var foundmember identity.Identity
	searchFilter := "General"
	tflog.Info(ctx, fmt.Sprintf("Searching for member: %s", name))
	var response, error = s.client.ReadIdentities(ctx, identity.ReadIdentitiesArgs{FilterValue: &name, SearchFilter: &searchFilter})
	if error != nil {
		error = fmt.Errorf("failed to read identities from azure devops: %w", error)
		return &identity.Identity{}, error
	}

	if len(*response) > 0 {
		foundmember = (*response)[0]
	} else {
		error = fmt.Errorf("failed to find identity %s in azure devops: %w", name, error)
		return &identity.Identity{}, error
	}

	return &foundmember, nil
}

func (s *IdentityService) GetIdentitiesByDescriptor(ctx context.Context, descriptor *string) (*[]identity.Identity, error) {
	var foundmembers []identity.Identity
	tflog.Info(ctx, fmt.Sprintf("Searching for descriptor: %s", *descriptor))
	var response, error = s.client.ReadIdentities(ctx, identity.ReadIdentitiesArgs{Descriptors: descriptor})
	if error != nil {
		error = fmt.Errorf("failed to read identities from azure devops: %w", error)
		return &[]identity.Identity{}, error
	}

	if len(*response) > 0 {
		foundmembers = *response
	} else {
		error = fmt.Errorf("failed to find identities with descriptor %s in azure devops: %w", *descriptor, error)
		return &[]identity.Identity{}, error
	}

	return &foundmembers, nil
}

func (s *IdentityService) GetGroup(ctx context.Context, name string) (*identity.Identity, error) {
	recurse := true
	var response, error = s.client.ListGroups(ctx, identity.ListGroupsArgs{Recurse: &recurse})
	if error != nil {
		error = fmt.Errorf("failed to list groups from azure devops: %w", error)
		return &identity.Identity{}, error
	}

	var foundGroup identity.Identity

	for _, group := range *response {
		if group.ProviderDisplayName != nil && *group.ProviderDisplayName == name {
			foundGroup = group
			break
		}
	}

	if (identity.Identity{}) == foundGroup {
		return &identity.Identity{}, fmt.Errorf("group %s not found", name)
	}

	return &foundGroup, nil
}

func (s *IdentityService) GetGroupMembers(ctx context.Context, name string) (*[]identity.Identity, error) {

	var foundGroup, err = s.GetGroup(ctx, name)
	if err != nil {
		return &[]identity.Identity{}, err
	}
	var foundGroupId = foundGroup.Id.String()
	var response, error = s.client.ReadMembers(ctx, identity.ReadMembersArgs{ContainerId: &foundGroupId})
	if error != nil {
		error = fmt.Errorf("failed to get members of group %s from azure devops: %w", name, error)
		return &[]identity.Identity{}, error
	}

	memberDescriptorsCombined := strings.Join(*response, ",")
	members, err := s.GetIdentitiesByDescriptor(ctx, &memberDescriptorsCombined)
	if err != nil {
		return &[]identity.Identity{}, err
	}
	var validMembers []identity.Identity
	for _, member := range *members {
		if member.CustomDisplayName != nil {
			validMembers = append(validMembers, member)
		}
	}

	slices.SortFunc(validMembers, func(i, j identity.Identity) int {
		return strings.Compare(*i.CustomDisplayName, *j.CustomDisplayName)
	})

	return &validMembers, nil
}

func (s *IdentityService) AddMemberToGroup(ctx context.Context, group *identity.Identity, member *identity.Identity) error {
	containerId := group.Id.String()
	memberId := member.Id.String()
	memberArgs := identity.AddMemberArgs{
		ContainerId: &containerId,
		MemberId:    &memberId,
	}
	_, err := s.client.AddMember(ctx, memberArgs)
	if err != nil {
		return fmt.Errorf("failed to add member %s to group: %s: %w", *member.ProviderDisplayName, *group.ProviderDisplayName, err)
	}
	return nil
}

func (s *IdentityService) RemoveMemberFromGroup(ctx context.Context, group *identity.Identity, member *identity.Identity) error {
	containerId := group.Id.String()
	memberId := member.Id.String()
	memberArgs := identity.RemoveMemberArgs{
		ContainerId: &containerId,
		MemberId:    &memberId,
	}
	_, err := s.client.RemoveMember(ctx, memberArgs)
	if err != nil {
		return fmt.Errorf("failed to remove member %s from group: %s: %w", *member.CustomDisplayName, *group.CustomDisplayName, err)
	}
	return nil
}

func (s *IdentityService) CreateGroup(ctx context.Context, name string, description string, members *[]identity.Identity) (*[]identity.Identity, error) {

	var group identity.Identity
	group.ProviderDisplayName = &name
	group.Description = &description

	var response, err = s.client.CreateGroups(ctx, identity.CreateGroupsArgs{)

	return response, nil
}
