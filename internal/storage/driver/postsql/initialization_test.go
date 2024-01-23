package postsql_test

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	maxTestDbAccessRetries = 20
)

func TestInitialization(t *testing.T) {

	t.Skip()
	t.Run("Should initialize database when schema not applied", func(t *testing.T) {
		cleanup, cfg, err := storage.InitTestDB(t)
		defer cleanup()
		require.NoError(t, err)

		// when
		connection, err := postsql.InitializeDatabase(cfg.ConnectionURL(), maxTestDbAccessRetries, logrus.New())
		require.NoError(t, err)
		require.NotNil(t, connection)
	})

	t.Run("Should return error when failed to connect to the database with bad connection string", func(t *testing.T) {
		// given
		connString := "bad connection string"

		// when
		connection, err := postsql.InitializeDatabase(connString, 3, logrus.New())

		// then
		assert.Error(t, err)
		assert.Nil(t, connection)
	})
}
