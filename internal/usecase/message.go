package usecase

import "github.com/wtitdn/renew_video/internal/repo"

type MesService struct{ Repo *repo.MesRepository }

func NewMesService(Repo *repo.MesRepository) *MesService { return &MesService{Repo: Repo} }
