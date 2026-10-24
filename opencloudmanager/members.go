package opencloudmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Role string

const (
	RoleViewer  Role = "viewer"
	RoleEditor  Role = "editor"
	RoleManager Role = "manager"
)

type Member struct {
	PermissionID string   `json:"permissionId"`
	UserID       string   `json:"userId,omitempty"`
	GroupID      string   `json:"groupId,omitempty"`
	DisplayName  string   `json:"displayName,omitempty"`
	RoleIDs      []string `json:"roles,omitempty"`
}

type RoleDefinition struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

type identity struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type identitySet struct {
	User  *identity `json:"user"`
	Group *identity `json:"group"`
}

type identitySets []identitySet

func (sets *identitySets) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*sets = nil
		return nil
	}
	if data[0] == '[' {
		return json.Unmarshal(data, (*[]identitySet)(sets))
	}
	var single identitySet
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	*sets = []identitySet{single}
	return nil
}

type permission struct {
	ID                  string       `json:"id"`
	Roles               []string     `json:"roles"`
	GrantedToV2         identitySets `json:"grantedToV2"`
	GrantedToIdentities identitySets `json:"grantedToIdentities"`
}

type permissionList struct {
	Value []permission     `json:"value"`
	Roles []RoleDefinition `json:"@libre.graph.permissions.roles.allowedValues"`
}

func (client *Client) ListMembers(ctx context.Context, driveID string) ([]Member, error) {
	var permissions permissionList
	if err := client.request(ctx, http.MethodGet, spacePermissionsPath(driveID), nil, &permissions); err != nil {
		return nil, err
	}
	members := make([]Member, 0, len(permissions.Value))
	for _, permission := range permissions.Value {
		identities := permission.GrantedToV2
		if len(identities) == 0 {
			identities = permission.GrantedToIdentities
		}
		member := Member{PermissionID: permission.ID, RoleIDs: permission.Roles}
		for _, identity := range identities {
			switch {
			case identity.User != nil:
				member.UserID = identity.User.ID
				member.DisplayName = identity.User.DisplayName
			case identity.Group != nil:
				member.GroupID = identity.Group.ID
				member.DisplayName = identity.Group.DisplayName
			}
			if member.UserID != "" || member.GroupID != "" {
				members = append(members, member)
				break
			}
		}
	}
	return members, nil
}

func (client *Client) ListAssignableRoles(ctx context.Context, driveID string) ([]RoleDefinition, error) {
	var permissions permissionList
	if err := client.request(ctx, http.MethodGet, spacePermissionsPath(driveID), nil, &permissions); err != nil {
		return nil, err
	}
	return permissions.Roles, nil
}

func (client *Client) InviteUser(ctx context.Context, driveID, userID string, role Role) error {
	if strings.TrimSpace(userID) == "" {
		return errors.New("OpenCloud user ID is required")
	}
	roleID, err := client.roleID(role)
	if err != nil {
		return err
	}
	input := struct {
		Recipients []struct {
			ObjectID string `json:"objectId"`
			Type     string `json:"@libre.graph.recipient.type"`
		} `json:"recipients"`
		Roles []string `json:"roles"`
	}{Roles: []string{roleID}}
	input.Recipients = append(input.Recipients, struct {
		ObjectID string `json:"objectId"`
		Type     string `json:"@libre.graph.recipient.type"`
	}{ObjectID: userID, Type: "user"})
	path := "/graph/v1beta1/drives/" + url.PathEscape(driveID) + "/root/invite"
	return client.request(ctx, http.MethodPost, path, input, nil)
}

func (client *Client) EnsureMember(ctx context.Context, driveID, userID string, role Role) (bool, error) {
	members, err := client.ListMembers(ctx, driveID)
	if err != nil {
		return false, err
	}
	for _, member := range members {
		if member.UserID == userID {
			return false, nil
		}
	}
	if err := client.InviteUser(ctx, driveID, userID, role); err != nil {
		return false, err
	}
	return true, nil
}

func (client *Client) SetMemberRole(ctx context.Context, driveID, permissionID string, role Role) error {
	roleID, err := client.roleID(role)
	if err != nil {
		return err
	}
	input := struct {
		Roles []string `json:"roles"`
	}{Roles: []string{roleID}}
	path := spacePermissionsPath(driveID) + "/" + url.PathEscape(permissionID)
	return client.request(ctx, http.MethodPatch, path, input, nil)
}

func (client *Client) RemoveMember(ctx context.Context, driveID, permissionID string) error {
	if strings.TrimSpace(permissionID) == "" {
		return errors.New("OpenCloud permission ID is required")
	}
	path := spacePermissionsPath(driveID) + "/" + url.PathEscape(permissionID)
	return client.request(ctx, http.MethodDelete, path, nil, nil)
}

func (client *Client) RemoveUser(ctx context.Context, driveID, userID string) error {
	members, err := client.ListMembers(ctx, driveID)
	if err != nil {
		return err
	}
	for _, member := range members {
		if member.UserID == userID {
			return client.RemoveMember(ctx, driveID, member.PermissionID)
		}
	}
	return fmt.Errorf("%w: %q", ErrNotMember, userID)
}

func (client *Client) roleID(role Role) (string, error) {
	var roleID string
	switch role {
	case RoleViewer:
		roleID = client.roles.Viewer
	case RoleEditor:
		roleID = client.roles.Editor
	case RoleManager:
		roleID = client.roles.Manager
	default:
		return "", fmt.Errorf("unsupported OpenCloud role %q", role)
	}
	if roleID == "" {
		return "", fmt.Errorf("OpenCloud role %q is not configured", role)
	}
	return roleID, nil
}

func spacePermissionsPath(driveID string) string {
	return "/graph/v1beta1/drives/" + url.PathEscape(driveID) + "/root/permissions"
}
