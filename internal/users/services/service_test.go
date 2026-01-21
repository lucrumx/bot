package services

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lucrumx/bot/internal/models"
)

type fakeUserRepo struct {
	user         *models.User
	createCalled bool
	createErr    error
}

func (f *fakeUserRepo) Create(user *models.User) error {
	f.createCalled = true
	f.user = user
	return f.createErr
}

func (f *fakeUserRepo) GetByID(_ uint) (*models.User, error) {
	return f.user, nil
}

func (f *fakeUserRepo) GetByEmail(_ string) (*models.User, error) {
	return f.user, nil
}

func TestUsersService_CreateUserSuccess(t *testing.T) {
	tests := []struct {
		name         string
		repo         *fakeUserRepo
		createCalled bool
		expectErr    bool
	}{
		{name: "Create user success", repo: &fakeUserRepo{}, createCalled: true},
		{
			name: "Create user fail",
			repo: &fakeUserRepo{
				createErr: errors.New("some error"),
			},
			createCalled: true,
			expectErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := Create(tt.repo)

			email := "some@email.com"
			password := "123456"

			user, err := srv.CreateUser(email, password)

			assert.True(t, tt.repo.createCalled, "Create method should be called")

			if tt.expectErr != true {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.NotEqual(t, user.PasswordHash, password)
			} else {
				assert.Error(t, err)
				assert.Nil(t, user)
			}
		})
	}
}
