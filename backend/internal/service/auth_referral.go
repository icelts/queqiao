package service

import (
	"context"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
)

type invitationBinding struct {
	inviterID  *int64
	redeemCode *RedeemCode
}

func (s *AuthService) ValidateInvitationCode(ctx context.Context, code string) error {
	if strings.TrimSpace(code) == "" {
		return ErrInvitationCodeInvalid
	}
	_, err := s.resolveInvitationBinding(ctx, code)
	return err
}

func (s *AuthService) resolveInvitationBinding(ctx context.Context, code string) (*invitationBinding, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, ErrInvitationCodeInvalid
	}

	referralBinding, err := s.resolveReferralBinding(ctx, code)
	if err != nil {
		return nil, ErrServiceUnavailable
	}
	if referralBinding != nil {
		return referralBinding, nil
	}

	redeemCode, err := s.redeemRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, ErrInvitationCodeInvalid
	}
	if redeemCode.Type != RedeemTypeInvitation || redeemCode.Status != StatusUnused {
		return nil, ErrInvitationCodeInvalid
	}
	return &invitationBinding{redeemCode: redeemCode}, nil
}

func (s *AuthService) resolveReferralBinding(ctx context.Context, code string) (*invitationBinding, error) {
	code = strings.TrimSpace(code)
	if code == "" || s.entClient == nil {
		return nil, nil
	}

	inviter, err := s.entClient.User.Query().
		Where(
			dbuser.ReferralCodeEQ(code),
			dbuser.StatusEQ(StatusActive),
			dbuser.DeletedAtIsNil(),
		).
		Only(ctx)
	if err == nil {
		return &invitationBinding{inviterID: &inviter.ID}, nil
	}
	if dbent.IsNotFound(err) {
		return nil, nil
	}
	return nil, err
}

func (s *AuthService) applyInvitationBinding(user *User, binding *invitationBinding) {
	if user == nil || binding == nil {
		return
	}
	user.InviterID = binding.inviterID
}

func (s *AuthService) markInvitationBindingUsed(ctx context.Context, binding *invitationBinding, userID int64) error {
	if binding == nil || binding.redeemCode == nil {
		return nil
	}
	return s.redeemRepo.Use(ctx, binding.redeemCode.ID, userID)
}
