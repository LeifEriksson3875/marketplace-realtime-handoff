package marketplacechat

import (
	"context"
	"errors"
	"fmt"
)

var ErrHandoffNotReady = errors.New("handoff requires a paid order and a ready seller asset")

type HandoffRequest struct {
	OrderID    string `json:"order_id"`
	SellerID   string `json:"seller_id"`
	BuyerID    string `json:"buyer_id"`
	AssetID    string `json:"asset_id"`
	Paid       bool   `json:"paid"`
	AssetReady bool   `json:"asset_ready"`
}

type HandoffResult struct {
	Channel string `json:"channel"`
	Token   string `json:"token"`
	State   string `json:"state"`
}

type Realtime interface {
	Publish(context.Context, PublishRequest, string) error
	IssueToken(context.Context, TokenRequest, string) (Token, error)
}

type HandoffService struct{ Realtime Realtime }

func (s HandoffService) Complete(ctx context.Context, request HandoffRequest) (HandoffResult, error) {
	if request.OrderID == "" || request.SellerID == "" || request.BuyerID == "" || request.AssetID == "" {
		return HandoffResult{}, errors.New("order, seller, buyer, and asset are required")
	}
	if !request.Paid || !request.AssetReady {
		return HandoffResult{}, ErrHandoffNotReady
	}
	channel := "marketplace:order:" + request.OrderID
	if err := s.Realtime.Publish(ctx, PublishRequest{
		Channel: channel,
		Event:   "order.handoff.ready",
		Data: map[string]string{
			"order_id":  request.OrderID,
			"asset_id":  request.AssetID,
			"seller_id": request.SellerID,
		},
		AccountID: request.SellerID,
	}, fmt.Sprintf("handoff-%s-%s", request.OrderID, request.AssetID)); err != nil {
		return HandoffResult{}, err
	}
	token, err := s.Realtime.IssueToken(ctx, TokenRequest{
		ClientID:     request.BuyerID,
		Channels:     []string{channel},
		Capabilities: []string{"subscribe"},
		TTLSeconds:   900,
	}, "buyer-token-"+request.OrderID+"-"+request.BuyerID)
	if err != nil {
		return HandoffResult{}, err
	}
	return HandoffResult{Channel: channel, Token: token.Token, State: "ready_for_buyer"}, nil
}
