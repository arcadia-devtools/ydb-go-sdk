package iam

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"

	"github.com/yandex-cloud/ydb-go-sdk/auth/iam/internal/genproto/iam/v1"
)

func TestGRPCCreateToken(t *testing.T) {
	const (
		token = "foo"
	)
	expires := time.Unix(0, 0)
	exp, err := ptypes.TimestampProto(expires)
	if err != nil {
		t.Fatal(err)
	}
	s := StubTokenService{
		OnCreate: func(ctx context.Context, req *v1.CreateIamTokenRequest) (
			res *v1.CreateIamTokenResponse, err error,
		) {
			return &v1.CreateIamTokenResponse{
				IamToken:  token,
				ExpiresAt: exp,
			}, nil
		},
	}
	addr, stop, err := s.ListenAndServe()
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	gt := grpcTransport{
		endpoint: addr.String(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tk, e, err := gt.CreateToken(ctx, "jwt")
	if act, exp := e, expires; !act.Equal(exp) {
		t.Errorf("unexpected expiration time: %v; want %v", act, exp)
	}
	if act, exp := tk, token; act != exp {
		t.Errorf("unexpected token: %q; want %q", act, exp)
	}
}

type StubTokenService struct {
	OnCreate func(context.Context, *v1.CreateIamTokenRequest) (*v1.CreateIamTokenResponse, error)
}

func (s *StubTokenService) ListenAndServe() (
	addr net.Addr,
	stop func() error,
	err error,
) {
	ln, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		return nil, nil, err
	}

	srv := grpc.NewServer()
	v1.RegisterIamTokenServiceServer(srv, s)

	serve := make(chan error)
	go func() { serve <- srv.Serve(ln) }()

	return ln.Addr(), func() error {
		srv.GracefulStop()
		return <-serve
	}, nil
}

func (s StubTokenService) Create(ctx context.Context, req *v1.CreateIamTokenRequest) (res *v1.CreateIamTokenResponse, err error) {
	if f := s.OnCreate; f != nil {
		return f(ctx, req)
	}
	return nil, fmt.Errorf("stub: not implemented")
}