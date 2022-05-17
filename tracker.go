package main

import (
	"crypto/tls"
	"encoding/json"
	"github.com/Pie-Messaging/core/pie"
	"github.com/Pie-Messaging/core/pie/routing"
	"math/big"
	"net"
	"time"
)

type Tracker struct {
	id           *big.Int
	listenAddr   string
	addr         []string
	cert         *tls.Certificate
	server       *pie.Server
	routingTable routing.Table
}

type ClientSession struct {
	session     *pie.Session
	peerTracker *routing.Tracker
}

type PeerTracker struct {
	ID   []byte
	Addr string
}

func (t *Tracker) init(cert *tls.Certificate, dbTrackers []*PeerTracker) {
	t.id = (&big.Int{}).SetBytes(pie.HashBytes(cert.Certificate[0], pie.IDLen))
	t.cert = cert
	trackers := make([]*routing.Tracker, len(dbTrackers))
	for i, dbTracker := range dbTrackers {
		tracker := &routing.Tracker{
			ID: (&big.Int{}).SetBytes(dbTracker.ID),
		}
		err := tracker.SetAddrStr(dbTracker.Addr)
		if err != nil {
			logger.Fatalln(err)
		}
		trackers[i] = tracker
	}
	t.routingTable.Init(ctx, trackers)
	t.bootstrap()
}

func (t *Tracker) bootstrap() {
	t.routingTable.FindTracker(ctx, t.id, pie.Alpha, recvTimeout)
}

func (t *Tracker) run(startupChan chan int) {
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*t.cert},
		NextProtos:   []string{pie.UserTLSProto, pie.TrackerTLSProto},
	}
	// TODO: token store
	server, err := pie.ListenNet(t.listenAddr, tlsConfig, nil)
	if err != nil {
		logger.Fatalln(err)
		return
	}
	startupChan <- server.Listener.Addr().(*net.UDPAddr).Port
	t.server = server
	for {
		session, err := server.AcceptSession(ctx)
		if err != nil {
			return
		}
		switch session.Session.ConnectionState().TLS.NegotiatedProtocol {
		case pie.UserTLSProto:
			ClientSession := ClientSession{session: session}
			go ClientSession.handleSession()
		case pie.TrackerTLSProto:
			trackerSession := ClientSession{session: session}
			go trackerSession.handleTrackerSession()
		}
	}
}

func (s *ClientSession) handleSession() {
	defer s.session.Close(pie.SessErrNoReason)
	logger.Debugln("Accepted new session:", s.session.Session.RemoteAddr())
	for {
		stream, err := s.session.AcceptStream(ctx, nil)
		if err != nil {
			logger.Debugln("Stream closed:", s.session.Session.RemoteAddr())
			return
		}
		go s.handleStream(stream)
	}
}

func (s *ClientSession) handleTrackerSession() {
	defer s.session.Close(pie.SessErrNoReason)
	id, err := s.verifyCert()
	if err != nil {
		return
	}
	if thisTracker.routingTable.GetTracker(id) == nil {
		go func() {
			addr, err := json.Marshal([]string{s.session.Session.RemoteAddr().String()})
			if err != nil {
				logger.Errorln("Failed to marshal addr:", err)
				return
			}
			tracker := &PeerTracker{ID: id, Addr: string(addr)}
			err = db.FirstOrCreate(tracker).Error
			if err != nil {
				logger.Errorln("Failed to insert into database:", err)
			}
		}()
	}
	s.peerTracker = &routing.Tracker{ID: (&big.Int{}).SetBytes(id), Addr: []string{s.session.Session.RemoteAddr().String()}}
	thisTracker.routingTable.AddTracker(s.peerTracker)
	for {
		stream, err := s.session.AcceptStream(ctx, nil)
		if err != nil {
			return
		}
		go s.handleStream(stream)
	}
}

func (s *ClientSession) verifyCert() ([]byte, error) {
	stream, err := s.session.AcceptStream(ctx, nil)
	if err != nil {
		return nil, err
	}
	message, err := stream.RecvMessage(time.Now().Add(recvTimeout))
	if err != nil {
		return nil, err
	}
	if clientCertReq := message.GetClientCertReq(); clientCertReq != nil {
		if certDER := clientCertReq.GetCertDer(); certDER != nil {
			if serverCertSign := clientCertReq.GetServerCertSign(); serverCertSign == nil {
				if err = thisTracker.server.VerifyClientCert(certDER, serverCertSign); err != nil {
					return nil, err
				}
				id := pie.HashBytes(certDER, pie.IDLen)
				return id, err
			}
		}
	}
	return nil, pie.ErrInvalidMsg
}

func (s *ClientSession) handleStream(stream *pie.Stream) {
	for {
		message, err := stream.RecvMessage(time.Time{})
		if err != nil {
			return
		}
		shouldContinue := s.handleMessage(stream, message)
		if !shouldContinue {
			return
		}
	}
}
