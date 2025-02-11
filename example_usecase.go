package module

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/delos-co/ENL-backend/core/entity"
	"github.com/delos-co/ENL-backend/payload"
	"github.com/delos-co/ENL-backend/pkg/common/helpers"
	"github.com/delos-co/ENL-backend/repository/project"

	ctxUtil "github.com/delos-co/ENL-backend/pkg/context"
)

type ProjectUsecase struct {
	ProjectRepo *project.ProjectRepo
}

func NewProjectUsecase(
	projectRepo *project.ProjectRepo,
) *ProjectUsecase {
	return &ProjectUsecase{
		ProjectRepo: projectRepo,
	}
}

const PROJECT string = "project"

func (w *ProjectUsecase) GetProjectList(
	ctx context.Context,
	filter *payload.ProjectFilter,
	pagination *payload.Pagination,
	sorting *payload.Sorting,
) ([]entity.ProjectEnt, error) {
	if pagination.PageSize == 0 {
		pagination.PageSize = payload.DefaultPageSize
	}
	if sorting.Direction == "" {
		sorting.Direction = payload.DefaultDirection
	}

	// If on first page, get the total items
	if pagination.Page == 1 {
		total, err := w.ProjectRepo.CountProjectTotalItem(ctx, filter)
		if err != nil {
			return []entity.ProjectEnt{}, errHandler(err, fmt.Sprintf(payload.ErrDataNotFound, PROJECT))
		}
		pagination.SetTotalItem(total)
		pagination.SetTotalPage(total)
	}

	resp, err := w.ProjectRepo.GetProjectList(ctx, pagination, sorting, filter)
	if err != nil {
		return []entity.ProjectEnt{}, errHandler(err, fmt.Sprintf(payload.ErrDataNotFound, PROJECT))
	}

	return resp, nil

}

func (w *ProjectUsecase) AddProject(
	ctx context.Context,
	req payload.AddProjectReq,
) (string, error) {
	var (
		err       error
		clientCtx = ctxUtil.ParseContextToClientContext(ctx)
		dateNow   = time.Now().In(time.UTC)
	)

	// Generate project no
	projectNo, err := helpers.Generate(`[A-Z0-9]{10}`)
	if err != nil {
		return projectNo, errHandler(err, fmt.Sprintf(payload.ErrGenerateID, PROJECT))
	}

	// Create project
	err = w.ProjectRepo.CreateProject(ctx, &entity.ProjectEnt{
		ProjectNo: projectNo,
		Name:      req.Name,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		CreatedAt: dateNow,
		CreatedBy: clientCtx.Fullname,
	})
	if err != nil {
		return projectNo, errHandler(err, fmt.Sprintf(payload.ErrAddData, PROJECT))
	}

	return projectNo, nil
}

func (w *ProjectUsecase) GetProjectByProjectNo(
	ctx context.Context,
	projectNo string,
) (entity.ProjectEnt, error) {
	resp, err := w.ProjectRepo.GetProjectByProjectNo(ctx, projectNo)
	if err != nil {
		return resp, errHandler(err, fmt.Sprintf(payload.ErrDataNotFound, PROJECT))
	}

	return resp, nil
}

func (w *ProjectUsecase) UpdateProjectByProjectNo(
	ctx context.Context,
	projectNo string,
	req payload.UpdateProjectReq,
) error {
	var (
		err       error
		clientCtx = ctxUtil.ParseContextToClientContext(ctx)
		dateNow   = time.Now().In(time.UTC)
	)

	// Update project
	err = w.ProjectRepo.UpdateProjectByProjectNo(ctx, entity.ProjectEnt{
		ProjectNo: projectNo,
		Name:      req.Name,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		UpdatedAt: &dateNow,
		UpdatedBy: &clientCtx.Fullname,
	})
	if err != nil {
		return errHandler(err, fmt.Sprintf(payload.ErrUpdateData, PROJECT))
	}

	return nil
}

func (w *ProjectUsecase) DeleteProjectByProjectNo(
	ctx context.Context,
	projectNo string,
) error {
	projectId, err := w.ProjectRepo.GetProjectIdByProjectNo(ctx, projectNo)
	if err != nil {
		return errHandler(err, fmt.Sprintf(payload.ErrDataNotFound, PROJECT))
	}

	// Check if there is any assign on this project
	exist, err := w.ProjectRepo.CheckAssignOnThisProjectExist(ctx, projectId)
	if err != nil {
		return errHandler(err, fmt.Sprintf(payload.ErrDataNotFound, PROJECT))
	}

	// If there is any assign on this project, return error
	if exist {
		return errors.New(fmt.Sprintf(payload.ErrForbiddenDeleteData, PROJECT))
	}

	// Delete project
	return w.ProjectRepo.DeleteProject(ctx, projectNo)
}
