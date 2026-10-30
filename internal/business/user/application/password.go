package application

import "context"

// ChangePassword is the protocol-neutral input for replacing credentials.
type ChangePassword struct {
	UserID          string
	CurrentPassword string
	NewPassword     string
}

// ChangePassword loads one User, applies the credential behavior, and saves it.
func (a *Application) ChangePassword(ctx context.Context, cmd ChangePassword) error {
	user, err := a.repository.Get(ctx, cmd.UserID)
	if err != nil {
		return err
	}
	if err = user.ChangePassword(cmd.CurrentPassword, cmd.NewPassword); err != nil {
		return err
	}
	return a.repository.Save(ctx, user)
}
