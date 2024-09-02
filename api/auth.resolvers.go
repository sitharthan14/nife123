package api

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/internal/users"
	"github.com/nifetency/nife.io/pkg/jwt"
)

func (r *mutationResolver) CreateUser(ctx context.Context, input model.NewUser) (string, error) {
	var user users.User
	user.Email = input.Email
	user.Password = input.Password
	user.Create()
	token, err := jwt.GenerateAccessToken(user.Email, "", false, user.FirstName, user.LastName, user.CompanyName, user.RoleId, "")
	if err != nil {
		return "", err
	}
	return token, nil
}

func (r *mutationResolver) Login(ctx context.Context, input model.Login) (string, error) {
	var user users.User
	user.Email = input.Email
	user.Password = input.Password
	correct, err := user.Authenticate()
	if err != nil {
		return "", err
	}
	if !correct {
		return "", &users.WrongEmailOrPasswordError{}
	}
	token, err := jwt.GenerateAccessToken(user.Email, "", false, user.FirstName, user.LastName, user.CompanyName, user.RoleId, "")
	if err != nil {
		return "", err
	}
	return token, nil
}
