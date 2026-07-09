package pool

import (
	"time"

	"github.com/ddukki/chu-go/conn"
)

type PoolConfig struct {
	Addrs []string
	conn.Config

	MaxConns    int
	MinConns    int
	MaxIdle     int
	MaxLifetime time.Duration

	HealthCheckInterval time.Duration
	HealthCheckTimeout  time.Duration
}
