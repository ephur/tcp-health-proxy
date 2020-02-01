package main

import (
	"net"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// EchoConfig provides the configuration for an echo server
type EchoConfig struct {
	ListenAddr string
	ListenPort string
}

// Echo contains the definition of an echo object
type Echo struct {
	listener  *net.TCPListener
	ch        chan bool
	waitGroup *sync.WaitGroup
	config    EchoConfig
	closing   bool
	starting  bool
}

// Accepted contains an accepted connection, and error, to unblock accepting requests
type accepted struct {
	conn *net.TCPConn
	err  error
}

// NewEcho returns a new echo object
func NewEcho(addr string, port string) *Echo {
	logger.Trace("creating new echo service")
	e := &Echo{
		config: EchoConfig{ListenAddr: addr,
			ListenPort: port,
		},
	}
	return e
}

// Up either starts a new echo service, or returns the existing one if it's already running
func (e *Echo) Up() *Echo {
	logger.Trace("entering into echo up function")
	if e.listener != nil {
		logger.Trace("a listener already exists, returning existing")
		return e
	}
	if e.starting {
		logger.Debug("listener is starting, go away")
		return e
	}

	// setup new echo service if one is not already running
	addrResolved, err := net.ResolveTCPAddr("tcp", e.config.ListenAddr+":"+e.config.ListenPort)
	if err != nil {
		logger.WithError(err).Error("invalid listen address")
		os.Exit(1)
	}
	logger.WithFields(log.Fields{"address": e.config.ListenAddr, "port": e.config.ListenPort}).Infof("starting TCP server")

	listener, err := net.ListenTCP("tcp", addrResolved)
	if err != nil {
		logger.WithError(err).Error("could not start listener")
		os.Exit(1)
	}

	e.listener = listener
	e.ch = make(chan bool)
	logger.Trace("creating wait group on listener object")
	e.waitGroup = &sync.WaitGroup{}
	e.waitGroup.Add(1)
	logger.Trace("added 1 to listener group")
	go e.handle()
	logger.Trace("finished starting the handler")
	e.starting = false
	logger.Infof("TCP server startup complete")
	return e
}

// handle handles incoming connections, and closing them down when needed
func (e *Echo) handle() {
	logger.Trace("entering into the handle routine for new connections")
	defer e.waitGroup.Done()
	for {
		// when this channel reads it will trigger a graceful shutdown of the listener
		select {
		case <-e.ch:
			logger.Trace("channel triggered, closing the listener")
			e.listener.Close()
			logger.Trace("listener successfully closed")
			e.listener = nil
			logger.Trace("returning back from the handler")
			return
		default:
		}
		// need to set this to ensure listener will not hang on graceful stop
		logger.Trace("accepting TCP connections from listener")
		c := make(chan accepted, 1)
		go func() {
			e.listener.SetDeadline(time.Now().Add(1 * time.Second))
			connection, err := e.listener.AcceptTCP()
			c <- accepted{connection, err}
		}()
		select {
		case a := <-c:
			if a.err != nil {
				if netErr, ok := a.err.(net.Error); ok && netErr.Temporary() {
					continue
				}
				logger.WithError(a.err).Info("Error on accept TCP")
				continue
			}
			logger.Trace("adding one process to wait group, about to launch echo process subroutine")
			logger.WithFields(log.Fields{"souce": a.conn.RemoteAddr()}).Debug("accepted connection")
			e.waitGroup.Add(1)
			go e.process(a.conn)
		}
	}
}

// Down gracefully stops the echo service to avoid any issues with active connections
func (e *Echo) Down() *Echo {
	// if the listener is nil, it's already down so return
	if e.listener == nil {
		logger.Trace("listener is nil, shutdown already complete, returning")
		return e
	}
	// if it takes longer to shut down than get a status update we could try and close twice, that's bad
	if e.closing {
		logger.Trace("closing is true, shutdown already in progress, returning")
		return e
	}
	logger.Infof("gracefully stopping TCP server")
	e.closing = true
	logger.Trace("closing down the echo channel")
	close(e.ch)
	logger.Trace("beginning wait for on wait group: ", e.waitGroup)
	e.waitGroup.Wait()
	logger.Trace("finished waiting on group, proceeding to set state of echo server")
	e.closing = false
	e.waitGroup = nil
	e.ch = nil
	logger.Infof("graceful TCP server termination complete")
	return e
}

// Serve a connection by reading and writing what was read.  That's right, this
// is an echo service.  Stop reading and writing if anything is received on the
// service's channel but only after writing what was read.
func (e *Echo) process(conn *net.TCPConn) {
	logger.Trace("entering into echo process function")
	defer conn.Close()
	defer e.waitGroup.Done()
	for {
		// return and abort the connection when the channel gets a stop signal
		select {
		case <-e.ch:
			logger.Trace("channel case triggered, returning out of process")
			return
		default:
		}
		// set a timeout on each iteration to send idle connections away
		conn.SetDeadline(time.Now().Add(3 * time.Second))
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		buf := make([]byte, 4096)
		logger.Trace("starting to read from connection buffer")
		if _, err := conn.Read(buf); err != nil {
			// there was an error / timeout, abort the connection
			logger.WithError(err).Debug("error during connection read, returning")
			return
		}
		logger.Trace("reading done, starting to write reply")
		if _, err := conn.Write(buf); err != nil {
			// there was an error sending data, abort the connection
			logger.WithError(err).Debug("error during connection write, returning")
			return
		}
		logger.Trace("done writing, going to start again")
	}
}
