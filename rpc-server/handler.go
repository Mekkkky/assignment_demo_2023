package main

import (
	"context"
	"math/rand"
	"fmt"
	"strings"
	"time"
	"github.com/TikTokTechImmersion/assignment_demo_2023/rpc-server/kitex_gen/rpc"
)

// IMServiceImpl implements the last service interface defined in the IDL.
type IMServiceImpl struct{}

func (s *IMServiceImpl) Send(ctx context.Context, req *rpc.SendRequest) (*rpc.SendResponse, error) {
	if err := validateSendRequest(req); err != nil {
		return nil, err
	}
	
	timestamp := time.Now().Unix()
	message := &Message{
		Sender:    req.Message.GetSender(),
		Message:   req.Message.GetText(),
		Timestamp: timestamp,
	}

	roomID, err := getRoomID(req.Message.GetChat())
	if err != nil {
		return nil, err
	}

	err = rdb.SaveMessage(ctx, roomID, message)
	if err != nil {
		return nil, err
	}

	resp := rpc.NewSendResponse()
	resp.Code, resp.Msg = 0, "success"
	
	return resp, nil
}

func (s *IMServiceImpl) Pull(ctx context.Context, req *rpc.PullRequest) (*rpc.PullResponse, error) {
	roomID, err := getRoomID(req.GetChat())
	if err != nil {
		return nil, err
	}

	limit := int64(req.GetLimit())
	if limit == 0 {
		limit = 20 // Default limit
	}
	start := req.GetCursor()
	end := start + limit // Did not minus 1, for hasMore check later on

	messages, err := rdb.GetMessagesByRoomID(ctx, roomID, start, end, req.GetReverse())
	if err != nil {
		return nil, err
	}

	respMessages := make([]*rpc.Message, 0)
	var counter int64 = 0
	var nextCursor int64 = 0
	hasMore := false
	for _, msg := range messages {
		if counter + 1 > limit {
			hasMore = true
			nextCursor = end
			break // Do not return the last message
		}

		temp := &rpc.Message{
			Chat:     req.GetChat(),
			Text:     msg.Message,
			Sender:   msg.Sender,
			SendTime: msg.Timestamp,
		}
		respMessages = append(respMessages, temp)
		counter += 1
	}
	
	resp := rpc.NewPullResponse()
	resp.Code = 0
	resp.Msg = "success"
	resp.Messages = respMessages
	resp.HasMore = &hasMore
	resp.NextCursor = &nextCursor
	
	return resp, nil
}

func validateSendRequest(req *rpc.SendRequest) error {
	senders := strings.Split(req.Message.Chat, ":")
	if len(senders) != 2 {
		err := fmt.Errorf("Invalid Chat ID '%s', should be in the format of user1:user2", req.Message.GetChat())
		return err
	}
	sender1, sender2 := senders[0], senders[1]

	if req.Message.GetSender() != sender1 && req.Message.GetSender() != sender2 {
		err := fmt.Errorf("Sender '%s' is not in the chat room", req.Message.GetSender())
		return err
	}

	return nil
}

func getRoomID(chat string) (string, error) {
	var roomID string

	lowercaseString := strings.ToLower(chat)
	senders := strings.Split(lowercaseString, ":")
	if len(senders) != 2 {
		err := fmt.Errorf("Invalid Chat ID '%s', should be in the format of user1:user2", chat)
		return "", err
	}

	// Compare the sender and receiver alphabetically, and sort it ascending to form the roomID
	sender1, sender2 := senders[0], senders[1]
	comparison := strings.Compare(sender1, sender2); 
	if comparison == 1 {
		roomID = fmt.Sprintf("%s:%s", sender2, sender1)
	} else {
		roomID = fmt.Sprintf("%s:%s", sender1, sender2)
	}

	return roomID, nil
}
