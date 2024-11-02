package dbrepo

import (
	"database/sql"

	"github.com/karalarmehmet/surveillance/internal/config"
	"github.com/karalarmehmet/surveillance/internal/repository"
)

type postgresDBRepo struct {
	App *config.AppConfig
	DB  *sql.DB
}

// NewPostgresRepo creates the repository
func NewPostgresRepo(Conn *sql.DB, a *config.AppConfig) repository.DatabaseRepo {
	return &postgresDBRepo{
		App: a,
		DB:  Conn,
	}
}

type testDBRepo struct {
	App *config.AppConfig
	// DB kaldırılabilir eğer kullanılmıyorsa
}

// NewTestingRepo creates a repo with a dummy database for testing
func NewTestingRepo(a *config.AppConfig) repository.DatabaseRepo {
	return &testDBRepo{
		App: a,
	}
}
