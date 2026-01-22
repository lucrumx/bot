package services

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"github.com/lucrumx/bot/internal/models"
)

func TestUsersService_CreateUser(t *testing.T) {
	email := "some@email.com"
	password := "some-password"

	tests := []struct {
		name        string
		setupMock   func(m *MockUserRepo)
		expectedErr bool
	}{
		{
			name: "Create user success",
			setupMock: func(m *MockUserRepo) {
				m.EXPECT().Create(mock.MatchedBy(func(u *models.User) bool {
					return u.Email == email && bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil
				})).Return(nil)
			},
			expectedErr: false,
		},
		{
			name: "Create user fail",
			setupMock: func(m *MockUserRepo) {
				m.EXPECT().Create(mock.Anything).Return(errors.New("some error"))
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockedUserRepo := NewMockUserRepo(t)

			tt.setupMock(mockedUserRepo)
			srv := Create(mockedUserRepo)

			user, err := srv.CreateUser(email, password)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, user.Email, email)
			}

		})
	}
}
