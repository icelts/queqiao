//go:build unit

package service

import (
	"context"
	"database/sql"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newAuthServiceEntClient(t *testing.T) *dbent.Client {
	t.Helper()

	db, err := sql.Open("sqlite", "file:auth_referral_binding?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestAuthService_Register_BindsReferralCodeWhenInvitationModeDisabled(t *testing.T) {
	ctx := context.Background()
	client := newAuthServiceEntClient(t)

	inviter, err := client.User.Create().
		SetEmail("inviter@test.com").
		SetPasswordHash("hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetReferralCode("ABCD1234").
		Save(ctx)
	require.NoError(t, err)

	repo := &userRepoStub{nextID: 1001}
	svc := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyInvitationCodeEnabled: "false",
	}, nil)
	svc.entClient = client

	_, user, err := svc.RegisterWithVerification(ctx, "new-user@test.com", "password", "", "", "ABCD1234")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Len(t, repo.created, 1)
	require.NotNil(t, repo.created[0].InviterID)
	require.Equal(t, inviter.ID, *repo.created[0].InviterID)
}

func TestAuthService_Register_InvalidReferralCodeWhenInvitationModeDisabled_NoBlock(t *testing.T) {
	ctx := context.Background()
	client := newAuthServiceEntClient(t)

	repo := &userRepoStub{nextID: 1002}
	svc := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyInvitationCodeEnabled: "false",
	}, nil)
	svc.entClient = client

	_, user, err := svc.RegisterWithVerification(ctx, "new-user2@test.com", "password", "", "", "INVALIDCODE")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Len(t, repo.created, 1)
	require.Nil(t, repo.created[0].InviterID)
}
