package gh

import (
	"github.com/cli/go-gh/v2/pkg/repository"
)

type Repository = repository.Repository

// CurrentRepository returns the current repository.
func CurrentRepository() (Repository, error) {
	return repository.Current()
}
