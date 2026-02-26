package gh

import (
	"github.com/cli/go-gh/v2/pkg/repository"
)

// Repository is an alias for repository.Repository from the go-gh package.
type Repository = repository.Repository

// CurrentRepository returns the current repository.
func CurrentRepository() (*Repository, error) {
	repo, err := repository.Current()
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func ParseRepositoryPath(path string) (*Repository, error) {
	repo, err := repository.Parse(path)
	if err != nil {
		return nil, err
	}
	return &repo, nil
}
