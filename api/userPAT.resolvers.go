package api

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/internal/auth"
	"github.com/nifetency/nife.io/service"
)

func (r *mutationResolver) AddPat(ctx context.Context, input *model.UserPat) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return "", err
	}

	err = service.InsertUserPAT(user.ID, *input)
	if err != nil {
		return "", err
	}

	return "Successfully inserted", nil
}

func (r *mutationResolver) UpdatePat(ctx context.Context, input *model.UserPat) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return "", err
	}

	err = service.UpdateUserPAT(user.ID, *input)
	if err != nil {
		return "", err
	}
	return "Updated successfully", nil
}

func (r *mutationResolver) DeletePat(ctx context.Context, id *string) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return "", err
	}

	err = service.DeleteUserPAT(user.ID, *id)
	if err != nil {
		return "", err
	}

	return "Successfully deleted", nil
}

func (r *queryResolver) GetUserPat(ctx context.Context) ([]*model.GetUserPat, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return []*model.GetUserPat{}, fmt.Errorf("Access Denied")
	}

	userPat, err := service.GetUserPAT(user.ID)
	if err != nil {
		return []*model.GetUserPat{}, err
	}

	return userPat, nil
}
