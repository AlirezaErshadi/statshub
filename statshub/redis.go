package statshub

import (
	"github.com/garyburd/redigo/redis"
	"net"
	"time"
)

const (
	connectionPoolSize = 1000

	redisConnectTimeout = 10 * time.Second
	redisReadTimeout    = 10 * time.Second
	redisWriteTimeout   = 10 * time.Second
)

var (
	connPool = make(chan redis.Conn, 1000)
)

// redisConn is a redis.Conn that stops processing new commands after it
// encounters its first error.
type redisConn struct {
	orig redis.Conn
	err  error
}

type redisDialer func(addr string, connectTimeout time.Duration) (net.Conn, error)

// connectToRedis() connects to our cloud Redis server and authenticates
func connectToRedis(dial redisDialer) (conn redis.Conn, err error) {
	select {
	case c := <-connPool:
		// Use pooled connection
		return c, nil
	default:
		// Create new connection
		return doConnectToRedis(dial)
	}
}

func doConnectToRedis(dial redisDialer) (conn redis.Conn, err error) {
	var nconn net.Conn

	if nconn, err = dial(redisAddr, redisConnectTimeout); err != nil {
		return
	}

	conn = &redisConn{orig: redis.NewConn(nconn, redisReadTimeout, redisWriteTimeout)}

	_, err = conn.Do("AUTH", redisPassword)
	return
}

func (conn *redisConn) Close() error {
	// Return connection to pool
	connPool <- conn
	return nil
}

func (conn *redisConn) Err() error {
	return conn.err
}

func (conn *redisConn) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	if conn.err != nil {
		return nil, conn.err
	} else {
		reply, err = conn.orig.Do(commandName, args...)
		conn.err = err
		return
	}
}

func (conn *redisConn) Send(commandName string, args ...interface{}) (err error) {
	if conn.err != nil {
		return conn.err
	} else {
		err = conn.orig.Send(commandName, args...)
		conn.err = err
		return
	}
}

func (conn *redisConn) Flush() (err error) {
	if conn.err != nil {
		return conn.err
	} else {
		err = conn.orig.Flush()
		conn.err = err
		return
	}
}

func (conn *redisConn) Receive() (reply interface{}, err error) {
	if conn.err != nil {
		return nil, conn.err
	} else {
		reply, err = conn.orig.Receive()
		conn.err = err
		return
	}
}
