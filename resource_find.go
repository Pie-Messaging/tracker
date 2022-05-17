package main

import (
	"errors"
	"github.com/Pie-Messaging/core/pie"
	"github.com/Pie-Messaging/core/pie/pb"
	"gorm.io/gorm"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

func (s *ClientSession) handleFindResource(stream *pie.Stream, request *pb.FindResourceReq) {
	id := request.GetId()
	if len(id) != pie.IDLen {
		return
	}
	switch request.GetType() {
	case pb.ResourceType_USER:
		s.handleFindUser(stream, id)
	}
}

func (s *ClientSession) handleFindUser(stream *pie.Stream, id []byte) {
	idInt := (&big.Int{}).SetBytes(id)
	user := User{}
	start := time.Now()
	if err := db.Take(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			trackers := thisTracker.routingTable.GetNeighbors(idInt, pie.KSize)
			candidates := make([]*pb.Tracker, len(trackers))
			for i, tracker := range trackers {
				candidates[i] = &pb.Tracker{
					Id:   tracker.ID.Bytes(),
					Addr: tracker.Addr,
				}
			}
			_ = stream.SendMessage(&pb.NetMessage{
				Body: &pb.NetMessage_FindResourceRes{
					FindResourceRes: &pb.FindResourceRes{
						Status:            pb.Status_NOT_FOUND,
						CandidateTrackers: candidates,
					},
				},
			})
			return
		}
		return
	}
	logger.Debugln("Query user in", time.Since(start))
	var avatar []byte
	if user.HasAvatar {
		var err error
		start := time.Now()
		avatar, err = os.ReadFile(filepath.Join(filesDir, (*big.Int)(user.ID).Text(16)))
		if err != nil {
			logger.Errorln("Failed to read avatar:", err)
		}
		logger.Debugln("Read avatar in", time.Since(start))
	}
	_ = stream.SendMessage(&pb.NetMessage{Body: &pb.NetMessage_FindResourceRes{FindResourceRes: &pb.FindResourceRes{
		Status: pb.Status_OK,
		Resource: &pb.Resource{Resource: &pb.Resource_User{
			User: &pb.User{
				Id:        (*big.Int)(user.ID).Bytes(),
				Name:      user.Name,
				Email:     user.Email,
				Bio:       user.Bio,
				Avatar:    avatar,
				CertDer:   user.CertDER,
				Addresses: user.Addr,
			},
		}},
	}}})
}
