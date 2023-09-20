package testutils

import (
	"fmt"
	"log"
	"strconv"
	"testing"

	"github.com/jinzhu/gorm"
	// import database's driver
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/ory/dockertest/v3"
	dc "github.com/ory/dockertest/v3/docker"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gopkg.in/gormigrate.v1"

	"github.com/insolar/block-explorer/migrations"
)

// RunDBInDockerWithPortBindings start a container with binding port to host machine
func RunDBInDockerWithPortBindings(dbName, dbPassword string) (int, *dockertest.Pool, *dockertest.Resource, func()) {
	pool := createPool()
	hostPort := 10_000
	var resource *dockertest.Resource
	var err error
	err = pool.Retry(func() error {
		// increase hostPort until can create a container
		hostPort++
		resource, err = pool.RunWithOptions(&dockertest.RunOptions{
			Repository: "postgres",
			Tag:        "12",
			Env: []string{
				"POSTGRES_DB=" + dbName,
				"POSTGRES_PASSWORD=" + dbPassword,
			},
			PortBindings: map[dc.Port][]dc.PortBinding{
				"5432/tcp": {{HostIP: "", HostPort: strconv.Itoa(hostPort)}},
			},
		})
		return err
	})
	if err != nil {
		log.Panicf("Could not start resource: %s", err)
	}

	poolCleaner := createPoolCleaner(pool, resource)
	return hostPort, pool, resource, poolCleaner
}

func createPool() *dockertest.Pool {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	return pool
}

func createPoolCleaner(pool *dockertest.Pool, resource *dockertest.Resource) func() {
	poolCleaner := func() {
		// When you're done, kill and remove the container
		err := pool.Purge(resource)
		if err != nil {
			log.Printf("failed to purge docker pool: %s", err)
		}
	}
	return poolCleaner
}

func RunDBInDocker(dbName, dbPassword string) (*dockertest.Pool, *dockertest.Resource, func()) {
	var err error
	pool := createPool()

	resource, err := pool.Run(
		"postgres", "12",
		[]string{
			"POSTGRES_DB=" + dbName,
			"POSTGRES_PASSWORD=" + dbPassword,
		},
	)
	if err != nil {
		log.Panicf("Could not start resource: %s", err)
	}

	poolCleaner := createPoolCleaner(pool, resource)
	return pool, resource, poolCleaner
}

func SetupDB() (*gorm.DB, func(), error) {
	dbName := "test_db"
	dbPassword := "secret"
	pool, resource, poolCleaner := RunDBInDocker(dbName, dbPassword)
	dbURL := fmt.Sprintf("postgres://postgres:%s@localhost:%s/%s?sslmode=disable", dbPassword, resource.GetPort("5432/tcp"), dbName)

	var db *gorm.DB
	err := pool.Retry(func() error {
		var err error

		db, err = gorm.Open("postgres", dbURL)
		if err != nil {
			return err
		}
		err = db.Exec("select 1").Error
		return err
	})
	if err != nil {
		poolCleaner()
		return nil, nil, errors.Wrap(err, "Could not start postgres:")
	}

	dbCleaner := func() {
		err := db.Close()
		if err != nil {
			log.Printf("failed to purge docker pool: %s", err)
		}
	}
	cleaner := func() {
		dbCleaner()
		poolCleaner()
	}

	m := gormigrate.New(db, migrations.MigrationOptions(), migrations.Migrations())

	if err = m.Migrate(); err != nil {
		return nil, nil, errors.Wrap(err, "Could not migrate:")
	}

	return db, cleaner, nil
}

func TruncateTables(t testing.TB, db *gorm.DB, models []interface{}) {
	for _, m := range models {
		err := db.Delete(m).Error
		require.NoError(t, err)
	}
}
