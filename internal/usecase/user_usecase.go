package usecase

import "github.com/rifkifajarramadhani/golang-clean-architecture/internal/domain"

type UserRepository interface {
	Create(user *domain.User) error
	GetAll() ([]*domain.User, error)
	GetByID(id int) (*domain.User, error)
	Update(user *domain.User) error
	Delete(id int) error
}

type UserUsecase struct {
	repo UserRepository
}

func NewUserUsecase(r UserRepository) *UserUsecase {
	return &UserUsecase{
		repo: r,
	}
}

func (u *UserUsecase) CreateUser(user *domain.User) error {
	return u.repo.Create(user)
}

func (u *UserUsecase) GetAllUsers() ([]*domain.User, error) {
	return u.repo.GetAll()
}

func (u *UserUsecase) GetUserByID(id int) (*domain.User, error) {
	return u.repo.GetByID(id)
}

func (u *UserUsecase) UpdateUser(user *domain.User) error {
	return u.repo.Update(user)
}

func (u *UserUsecase) DeleteUser(id int) error {
	return u.repo.Delete(id)
}
