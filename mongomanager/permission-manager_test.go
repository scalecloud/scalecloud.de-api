package mongomanager

import "testing"

func TestContainsRole(t *testing.T) {
	tests := []struct {
		name         string
		seatRoles    []Role
		required     []Role
		wantContains bool
	}{
		{
			name:         "matching role grants permission",
			seatRoles:    []Role{RoleBilling},
			required:     []Role{RoleBilling},
			wantContains: true,
		},
		{
			name:         "owner implicitly has every role",
			seatRoles:    []Role{RoleOwner},
			required:     []Role{RoleAdministrator},
			wantContains: true,
		},
		{
			name:      "unrelated role does not grant permission",
			seatRoles: []Role{RoleUser},
			required:  []Role{RoleBilling},
		},
		{
			name:     "empty roles do not grant permission",
			required: []Role{RoleBilling},
		},
		{
			name:         "owner grants access when no roles are listed",
			seatRoles:    []Role{RoleOwner},
			wantContains: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			seat := Seat{Roles: testCase.seatRoles}
			if got := ContainsRole(seat, testCase.required); got != testCase.wantContains {
				t.Fatalf("ContainsRole(%v, %v) = %t, want %t", testCase.seatRoles, testCase.required, got, testCase.wantContains)
			}
		})
	}
}
