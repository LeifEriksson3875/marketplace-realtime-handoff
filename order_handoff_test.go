package marketplacechat

import (
	"context"
	"errors"
	"testing"
)

type realtimeSpy struct {
	published []PublishRequest
	tokens    []TokenRequest
}

func (s *realtimeSpy) Publish(_ context.Context, request PublishRequest, _ string) error {
	s.published = append(s.published, request)
	return nil
}

func (s *realtimeSpy) IssueToken(_ context.Context, request TokenRequest, _ string) (Token, error) {
	s.tokens = append(s.tokens, request)
	return Token{Token: "buyer-session-token"}, nil
}

func TestHandoffDecision(t *testing.T) {
	tests := []struct {
		name       string
		paid       bool
		assetReady bool
		wantReady  bool
	}{
		{name: "paid order and ready asset", paid: true, assetReady: true, wantReady: true},
		{name: "payment pending", paid: false, assetReady: true},
		{name: "asset pending", paid: true, assetReady: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spy := &realtimeSpy{}
			service := HandoffService{Realtime: spy}
			result, err := service.Complete(context.Background(), HandoffRequest{
				OrderID: "ord-42", SellerID: "seller-7", BuyerID: "buyer-9", AssetID: "asset-3",
				Paid: test.paid, AssetReady: test.assetReady,
			})
			if !test.wantReady {
				if !errors.Is(err, ErrHandoffNotReady) || len(spy.published) != 0 || len(spy.tokens) != 0 {
					t.Fatalf("pending handoff: err=%v publishes=%d tokens=%d", err, len(spy.published), len(spy.tokens))
				}
				return
			}
			if err != nil || result.State != "ready_for_buyer" || result.Channel != "marketplace:order:ord-42" {
				t.Fatalf("ready handoff: result=%+v err=%v", result, err)
			}
			if len(spy.published) != 1 || spy.published[0].Event != "order.handoff.ready" || len(spy.tokens) != 1 {
				t.Fatalf("unexpected calls: publishes=%+v tokens=%+v", spy.published, spy.tokens)
			}
		})
	}
}
