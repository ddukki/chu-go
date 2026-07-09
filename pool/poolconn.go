package pool

import (
	"github.com/jackc/puddle/v2"

	"github.com/ddukki/chu-go/conn"
)

type PoolConn struct {
	*conn.Conn
	addr string
	res  *puddle.Resource[*conn.Conn]
}

func (pc *PoolConn) Release() {
	pc.res.Release()
}

func (pc *PoolConn) Close() {
	pc.res.Destroy()
}

func (pc *PoolConn) Addr() string {
	return pc.addr
}
