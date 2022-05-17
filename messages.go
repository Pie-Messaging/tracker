package main

import (
	"github.com/Pie-Messaging/core/pie"
	"github.com/Pie-Messaging/core/pie/pb"
	"math/big"
	"net"
)

func (s *ClientSession) handleMessage(stream *pie.Stream, message *pb.NetMessage) bool {
	switch body := message.GetBody().(type) {
	case *pb.NetMessage_GetAddrReq:
		s.handleGetAddr(stream, body.GetAddrReq)
	case *pb.NetMessage_FindTrackerReq:
		s.handleFindTracker(stream, body.FindTrackerReq)
	case *pb.NetMessage_FindResourceReq:
		s.handleFindResource(stream, body.FindResourceReq)
	case *pb.NetMessage_PutResourceReq:
		s.handlePutResource(stream, body.PutResourceReq)
	}
	return false
}

func (s *ClientSession) handleGetAddr(stream *pie.Stream, request *pb.GetAddrReq) {
	_ = stream.SendMessage(&pb.NetMessage{
		Body: &pb.NetMessage_GetAddrRes{
			GetAddrRes: &pb.GetAddrRes{
				Addresses: []string{s.session.Session.RemoteAddr().(*net.UDPAddr).IP.String()},
			},
		},
	})
}

func (s *ClientSession) handleFindTracker(stream *pie.Stream, request *pb.FindTrackerReq) {
	targetIDInt := (&big.Int{}).SetBytes(request.GetId())
	var excludeID *big.Int
	if s.peerTracker != nil {
		excludeID = s.peerTracker.ID
	}
	result := thisTracker.routingTable.GetNeighbors(targetIDInt, pie.KSize, excludeID)
	response := make([]*pb.Tracker, 0, len(result))
	for _, tracker := range result {
		response = append(response, &pb.Tracker{
			Id:   tracker.ID.Bytes(),
			Addr: tracker.Addr,
		})
	}
	_ = stream.SendMessage(&pb.NetMessage{
		Body: &pb.NetMessage_FindTrackerRes{
			FindTrackerRes: &pb.FindTrackerRes{
				Status:     pb.Status_OK,
				Candidates: response,
			},
		},
	})
}
