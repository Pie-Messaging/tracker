package main

import (
	"github.com/Pie-Messaging/core/pie"
	"github.com/Pie-Messaging/core/pie/pb"
	"gorm.io/gorm/clause"
	"math/big"
	"os"
	"path/filepath"
)

func (s *ClientSession) handlePutResource(stream *pie.Stream, request *pb.PutResourceReq) {
	resource := request.GetResource()
	switch request.GetType() {
	case pb.ResourceType_USER:
		s.handlePutUser(stream, resource.GetUser())
	}
}

func (s *ClientSession) handlePutUser(stream *pie.Stream, user *pb.User) {
	userDB := &User{
		ID:        (*pie.ID)((&big.Int{}).SetBytes(user.GetId())),
		Name:      user.GetName(),
		Email:     user.GetEmail(),
		Bio:       user.GetBio(),
		HasAvatar: user.GetAvatar() != nil,
		CertDER:   user.GetCertDer(),
		Addr:      user.GetAddresses(),
	}
	avatarPath := filepath.Join(filesDir, (*big.Int)(userDB.ID).Text(16))
	if userDB.HasAvatar {
		err := os.WriteFile(avatarPath, user.GetAvatar(), 0644)
		if err != nil {
			logger.Errorln("Failed to write avatar file:", err)
			return
		}
	} else {
		err := os.Remove(avatarPath)
		if err != nil {
			logger.Errorln("Failed to remove avatar file:", err)
		}
	}
	if err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(userDB).Error; err != nil {
		logger.Errorln("Failed to create user:", err)
		return
	}
	logger.Debugf("Created user: %s Addr: %v DBAddr: %v", (*big.Int)(userDB.ID).Text(16), user.GetAddresses(), userDB.Addr)
	_ = stream.SendMessage(&pb.NetMessage{
		Body: &pb.NetMessage_PutResourceRes{
			PutResourceRes: &pb.PutResourceRes{
				Status: pb.Status_OK,
			},
		},
	})
	return
}
