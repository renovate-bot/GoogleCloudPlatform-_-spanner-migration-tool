// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-20.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cassandraclient

import (
	"fmt"
	"sync"
	"time"

	"github.com/gocql/gocql"
)

var clusterConfigMux sync.Mutex
var globalClusterConfig *gocql.ClusterConfig

// newCluster is declared as a global variable for easy mocking of gocql.NewCluster during tests.
var newCluster = gocql.NewCluster

var createSessionFromCluster = func(c *gocql.ClusterConfig) (GocqlSessionInterface, error) {
	session, err := c.CreateSession()
	if err != nil {
		return nil, err
	}
	return NewGocqlSessionImpl(session), nil
}

func GetOrCreateCassandraClusterClient(contactPoints []string, port int, keyspace, datacenter, user, password string) (CassandraClusterInterface, error) {
	clusterConfigMux.Lock()
	defer clusterConfigMux.Unlock()
	
	clusterCfg := newCluster(contactPoints...)
	clusterCfg.Keyspace = keyspace
	clusterCfg.Consistency = gocql.Quorum
	clusterCfg.Timeout = 10 * time.Second
	clusterCfg.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries: 3}
	if port > 0 {
		clusterCfg.Port = port
	}
	if user != "" {
		clusterCfg.Authenticator = gocql.PasswordAuthenticator{Username: user, Password: password}
	}
	if datacenter != "" {
		clusterCfg.PoolConfig.HostSelectionPolicy = gocql.DCAwareRoundRobinPolicy(datacenter)
	}
	globalClusterConfig = clusterCfg

	wrappedSession, err := createSessionFromCluster(globalClusterConfig)

	if err != nil {
		return nil, fmt.Errorf("failed to create Cassandra session: %w", err)
	}
	return &CassandraClusterImpl{session: wrappedSession}, nil
}
