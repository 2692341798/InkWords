package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type fakeRepository struct {
	user      *User
	saveErr   error
	saveCalls int
}

func (f *fakeRepository) CountByEmailOrUsername(ctx context.Context, email string, username string) (int64, error) {
	if f.user != nil && f.user.Email == email {
		return 1, nil
	}
	return 0, nil
}

func (f *fakeRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	if f.user == nil || f.user.Email != email {
		return nil, errors.New("not found")
	}
	clone := *f.user
	return &clone, nil
}

func (f *fakeRepository) GetByGithubIDOrEmail(ctx context.Context, githubID string, email string) (*User, error) {
	return nil, errors.New("unexpected call to GetByGithubIDOrEmail in email-login test")
}

func (f *fakeRepository) Create(ctx context.Context, user *User) error {
	return errors.New("unexpected call to Create in email-login test")
}

func (f *fakeRepository) Save(ctx context.Context, user *User) error {
	f.saveCalls++
	return f.saveErr
}

func TestLogin_ReturnsInternalErrorWhenFailedAttemptCannotBePersisted(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	require.NoError(t, err)
	repo := &fakeRepository{
		user:    &User{ID: uuid.New(), Email: "user@example.com", PasswordHash: string(hash)},
		saveErr: errors.New("database unavailable"),
	}

	_, _, err = NewService(repo).Login(context.Background(), "user@example.com", "wrong-password", "", "")

	require.ErrorContains(t, err, "persist failed login state")
	require.Equal(t, 1, repo.saveCalls)
}

func TestLogin_ClearsFailedAttemptsOnValidPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	require.NoError(t, err)
	repo := &fakeRepository{
		user: &User{
			ID:                  uuid.New(),
			Email:               "user@example.com",
			PasswordHash:        string(hash),
			FailedLoginAttempts: 2,
		},
	}

	_, _, err = NewService(repo).Login(context.Background(), "user@example.com", "correct-password", "", "")

	require.NoError(t, err)
	require.Equal(t, 1, repo.saveCalls)
}

func TestLogin_ReturnsInternalErrorWhenResetSaveFails(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	require.NoError(t, err)
	repo := &fakeRepository{
		user: &User{
			ID:                  uuid.New(),
			Email:               "user@example.com",
			PasswordHash:        string(hash),
			FailedLoginAttempts: 2,
		},
		saveErr: errors.New("database unavailable"),
	}

	_, _, err = NewService(repo).Login(context.Background(), "user@example.com", "correct-password", "", "")

	require.ErrorContains(t, err, "persist successful login state")
	require.Equal(t, 1, repo.saveCalls)
}
