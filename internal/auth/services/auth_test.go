package services

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/lucrumx/bot/internal/config"

	"github.com/lucrumx/bot/internal/models"
	"github.com/lucrumx/bot/internal/utils"
)

type fakeUserFinder struct {
	user *models.User
	err  error
}

func (f *fakeUserFinder) GetByEmail(_ string) (*models.User, error) {
	if f.user.Active == false {
		return nil, errors.New("user inactive")
	}

	return f.user, f.err
}

func TestAuthService_Login(t *testing.T) {
	t.Setenv("JWT_SECRET", "some-secret-key")

	const email = "some@email.com"
	const password = "some-password"
	passwordHash, _ := utils.GenPasswordHash(password)

	user := &models.User{
		Model: gorm.Model{
			ID: 1,
		},
		Active:       true,
		Email:        email,
		PasswordHash: passwordHash,
	}

	tests := []struct {
		name      string
		finder    UserFinder
		email     string
		password  string
		wantToken bool
	}{
		{
			name: "valid credentials",
			finder: &fakeUserFinder{
				user: user,
				err:  nil,
			},
			email:     email,
			password:  password,
			wantToken: true,
		},
		{
			name: "user not found",
			finder: &fakeUserFinder{
				user: user,
				err:  errors.New("user not found"),
			},
			email:    "wrong@email",
			password: password,
		},
		{
			name: "invalid password",
			finder: &fakeUserFinder{
				user: user,
				err:  errors.New("invalid password"),
			},
			email:    email,
			password: "wrong-password",
		},
		{
			name: "user inactive",
			finder: &fakeUserFinder{
				user: &models.User{
					Email:        email,
					PasswordHash: passwordHash,
					Active:       false,
				},
				err: errors.New("user inactive"),
			},
			email:    email,
			password: "wrong-password",
		},
	}

	cfg := &config.Config{
		HTTP: config.HTTPConfig{
			Auth: config.AuthConfig{
				JwtSecret:    "some-secret-key",
				JwtExpiresIn: 24,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := Create(tt.finder, cfg)

			token, err := srv.Login(tt.email, tt.password)
			if tt.wantToken {
				assert.NoError(t, err)
				assert.True(t, len(token) > 0)
			} else {
				assert.True(t, token == "")
				assert.Error(t, err)
			}
		})
	}
}
