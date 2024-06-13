package firebasemanager

import (
	"context"

	"firebase.google.com/go/v4/auth"
)

func (firebaseConnection *FirebaseConnection) InviteSeat(ctx context.Context, email string) error {
	_, err := firebaseConnection.GetUserByEmail(ctx, email)
	if err != nil {
		if auth.IsUserNotFound(err) {
			_, err := firebaseConnection.createUser(ctx, email)
			if err != nil {
				return err
			}
			err = firebaseConnection.sendInviteAndVerifyEMail(ctx, email)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	err = firebaseConnection.sendInviteEMail(email)
	if err != nil {
		return err
	}
	return nil
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

func (firebaseConnection *FirebaseConnection) sendInviteEMail(email string) error {
	// TODO: User already exits, user recives an email with the invite link where he just accepts the invite
	firebaseConnection.log.Error("Verification link sending implemetation missing: " + email)
	return nil
}

func (firebaseConnection *FirebaseConnection) sendInviteAndVerifyEMail(ctx context.Context, email string) error {
	client, err := firebaseConnection.firebaseApp.Auth(ctx)
	if err != nil {
		return err
	}
	link, err := client.EmailVerificationLink(ctx, email)
	if err != nil {
		return err
	}
	// TODO: Send you got invited email where the link verifies the email and also accepts the invite to the subscription
	firebaseConnection.log.Error("Verification link sending implemetation missing: " + link)
	return nil
}
