package firebasemanager

import (
	"context"
	"errors"

	"firebase.google.com/go/v4/auth"
)

func (firebaseConnection *FirebaseConnection) InviteSeat(ctx context.Context, email string) error {
	user, err := firebaseConnection.GetUserByEmail(ctx, email)
	if err != nil {
		if auth.IsUserNotFound(err) {
			_, err := firebaseConnection.createUser(ctx, email)
			if err != nil {
				return err
			}
			err = firebaseConnection.sendInviteVerifyEMail(ctx, email)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if user != nil {
		return errors.New("user already exists")
	} else {
		firebaseConnection.log.Warn("Should not happen: user is nil")
		return errors.New("user is nil")
	}
}

func (firebaseConnection *FirebaseConnection) GetUserByEmail(ctx context.Context, email string) (*auth.UserRecord, error) {
	client, err := firebaseConnection.firebaseApp.Auth(ctx)
	if err != nil {
		return nil, err
	}
	user, err := client.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (firebaseConnection *FirebaseConnection) createUser(ctx context.Context, email string) (*auth.UserRecord, error) {
	client, err := firebaseConnection.firebaseApp.Auth(ctx)
	if err != nil {
		return nil, err
	}
	params := (&auth.UserToCreate{}).Email(email).EmailVerified(false)
	user, err := client.CreateUser(ctx, params)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (firebaseConnection *FirebaseConnection) sendInviteVerifyEMail(ctx context.Context, email string) error {
	client, err := firebaseConnection.firebaseApp.Auth(ctx)
	if err != nil {
		return err
	}
	link, err := client.EmailVerificationLink(ctx, email)
	if err != nil {
		return err
	}
	// TODO: Send you got invited email with verification link
	firebaseConnection.log.Error("Verification link sending implemetation missing: " + link)
	return nil
}
