// Copyright 2016 Documize IntestConfigPublicLDAP. <legal@documize.com>. All rights reserved.
//
// This software (Documize Community Edition) is licensed under
// GNU AGPL v3 http://www.gnu.org/licenses/agpl-3.0.en.html
//
// You can operate outside the AGPL restrictions by purchasing
// Documize Enterprise Edition and obtaining a commercial license
// by contacting <sales@documize.com>.
//
// https://documize.com

package ldap

import (
	"fmt"
	"strings"
	"testing"

	lm "github.com/documize/community/model/auth"
	ld "gopkg.in/ldap.v2"
)

// Works against https://www.forumsys.com/tutorials/integration-how-to/ldap/online-ldap-test-server/

var testConfigPublicLDAP = lm.LDAPConfig{
	ServerType:               lm.ServerTypeLDAP,
	ServerHost:               "ldap.forumsys.com",
	ServerPort:               389,
	EncryptionType:           "none",
	BaseDN:                   "dc=example,dc=com",
	BindDN:                   "cn=read-only-admin,dc=example,dc=com",
	BindPassword:             "password",
	UserFilter:               "",
	GroupFilter:              "",
	AttributeUserRDN:         "uid",
	AttributeUserFirstname:   "givenName",
	AttributeUserLastname:    "sn",
	AttributeUserEmail:       "mail",
	AttributeUserDisplayName: "",
	AttributeUserGroupName:   "",
	AttributeGroupMember:     "uniqueMember",
}

func TestPublicLDAPServer_UserList(t *testing.T) {
	testConfigPublicLDAP.UserFilter = "(|(objectClass=person)(objectClass=user)(objectClass=inetOrgPerson))"
	testConfigPublicLDAP.GroupFilter = ""
	userAttrs := testConfigPublicLDAP.GetUserFilterAttributes()

	l, err := Connect(testConfigPublicLDAP)
	if err != nil {
		t.Error("Error: unable to dial LDAP server: ", err.Error())
		return
	}
	defer l.Close()

	// Authenticate with LDAP server using admin credentials.
	t.Log("Binding LDAP admin user")
	err = l.Bind(testConfigPublicLDAP.BindDN, testConfigPublicLDAP.BindPassword)
	if err != nil {
		t.Error("Error: unable to bind specified admin user to LDAP: ", err.Error())
		return
	}

	searchRequest := ld.NewSearchRequest(
		testConfigPublicLDAP.BaseDN,
		ld.ScopeWholeSubtree, ld.NeverDerefAliases, 0, 0, false,
		testConfigPublicLDAP.UserFilter,
		userAttrs,
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		t.Error("Error: unable to execute directory search: ", err.Error())
		return
	}

	t.Logf("LDAP search entries found: %d", len(sr.Entries))
	if len(sr.Entries) == 0 {
		t.Error("Received ZERO LDAP search entries")
		return
	}

	for _, entry := range sr.Entries {
		t.Logf("[%s] %s (%s %s) @ %s\n",
			entry.GetAttributeValue(testConfigPublicLDAP.AttributeUserRDN),
			entry.GetAttributeValue("cn"),
			entry.GetAttributeValue(testConfigPublicLDAP.AttributeUserFirstname),
			entry.GetAttributeValue(testConfigPublicLDAP.AttributeUserLastname),
			entry.GetAttributeValue(testConfigPublicLDAP.AttributeUserEmail))
	}
}

func TestPublicLDAPServer_Groups(t *testing.T) {
	testConfigPublicLDAP.UserFilter = ""
	testConfigPublicLDAP.GroupFilter = "(|(ou=mathematicians)(ou=chemists))"
	groupAttrs := testConfigPublicLDAP.GetGroupFilterAttributes()
	userAttrs := testConfigPublicLDAP.GetUserFilterAttributes()

	l, err := Connect(testConfigPublicLDAP)
	if err != nil {
		t.Error("Error: unable to dial LDAP server: ", err.Error())
		return
	}
	defer l.Close()

	// Authenticate with LDAP server using admin credentials.
	t.Log("Binding LDAP admin user")
	err = l.Bind(testConfigPublicLDAP.BindDN, testConfigPublicLDAP.BindPassword)
	if err != nil {
		t.Error("Error: unable to bind specified admin user to LDAP: ", err.Error())
		return
	}

	searchRequest := ld.NewSearchRequest(
		testConfigPublicLDAP.BaseDN,
		ld.ScopeWholeSubtree, ld.NeverDerefAliases, 0, 0, false,
		testConfigPublicLDAP.GroupFilter,
		groupAttrs,
		nil,
	)

	t.Log("LDAP search filter:", testConfigPublicLDAP.GroupFilter)
	sr, err := l.Search(searchRequest)
	if err != nil {
		t.Error("Error: unable to execute directory search: ", err.Error())
		return
	}

	t.Logf("LDAP search entries found: %d", len(sr.Entries))
	if len(sr.Entries) == 0 {
		t.Error("Received ZERO LDAP search entries")
		return
	}

	// Get list of group members per group found.
	for _, group := range sr.Entries {
		t.Log("Found group", group.DN)

		rawMembers := group.GetAttributeValues(testConfigPublicLDAP.AttributeGroupMember)
		if len(rawMembers) == 0 {
			t.Log("Error: group member attribute returned no users")
			continue
		}

		t.Logf("LDAP group contains %d members", len(rawMembers))

		for _, entry := range rawMembers {
			// get CN element from DN
			parts := strings.Split(entry, ",")
			if len(parts) == 0 {
				continue
			}
			filter := fmt.Sprintf("(%s)", parts[0])

			usr := ld.NewSearchRequest(
				testConfigPublicLDAP.BaseDN,
				ld.ScopeWholeSubtree, ld.NeverDerefAliases, 0, 0, false,
				filter,
				userAttrs,
				nil,
			)
			ue, err := l.Search(usr)
			if err != nil {
				t.Log("Error: unable to execute directory search for group member: ", err.Error())
				continue
			}

			if len(ue.Entries) > 0 {
				for _, ur := range ue.Entries {
					t.Logf("[%s] %s (%s %s) @ %s\n",
						ur.GetAttributeValue(testConfigPublicLDAP.AttributeUserRDN),
						ur.GetAttributeValue("cn"),
						ur.GetAttributeValue(testConfigPublicLDAP.AttributeUserFirstname),
						ur.GetAttributeValue(testConfigPublicLDAP.AttributeUserLastname),
						ur.GetAttributeValue(testConfigPublicLDAP.AttributeUserEmail))
				}
			} else {
				t.Log("group member search failed:", filter)
			}
		}
	}
}

func TestPublicLDAP_Authenticate(t *testing.T) {
	testConfigPublicLDAP.UserFilter = ""
	testConfigPublicLDAP.GroupFilter = ""
	userAttrs := testConfigPublicLDAP.GetUserFilterAttributes()

	l, err := Connect(testConfigPublicLDAP)
	if err != nil {
		t.Error("Error: unable to dial LDAP server: ", err.Error())
		return
	}
	defer l.Close()

	// Authenticate with LDAP server using admin credentials.
	t.Log("Binding LDAP admin user")
	err = l.Bind(testConfigPublicLDAP.BindDN, testConfigPublicLDAP.BindPassword)
	if err != nil {
		t.Error("Error: unable to bind specified admin user to LDAP: ", err.Error())
		return
	}

	username := "newton"
	password := "password"
	filter := fmt.Sprintf("(%s=%s)", testConfigPublicLDAP.AttributeUserRDN, username)

	searchRequest := ld.NewSearchRequest(
		testConfigPublicLDAP.BaseDN,
		ld.ScopeWholeSubtree, ld.NeverDerefAliases, 0, 0, false,
		filter,
		userAttrs,
		nil,
	)

	t.Log("LDAP search filter:", filter)
	sr, err := l.Search(searchRequest)
	if err != nil {
		t.Error("Error: unable to execute directory search: ", err.Error())
		return
	}

	if len(sr.Entries) == 0 {
		t.Error("Error: user not found")
		return
	}
	if len(sr.Entries) != 1 {
		t.Error("Error: too many users found during authentication")
		return
	}

	userdn := sr.Entries[0].DN

	// Bind as the user to verify their password
	err = l.Bind(userdn, password)
	if err != nil {
		t.Error("Error: invalid credentials", err.Error())
		return
	}

	t.Log("Authenticated", username)
}