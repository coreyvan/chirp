package node

import (
	"context"
	"errors"
	"testing"

	pb "github.com/coreyvan/chirp/protogen/meshtastic"
)

type fakeClient struct {
	infoResponses []*pb.FromRadio
	infoErr       error

	sendErr     error
	sendCalls   int
	sendMessage string
	sendTo      int64
	sendChannel int64

	ownerErr   error
	ownerCalls int
	ownerName  string

	modemErr   error
	modemCalls int
	modemMode  string

	locationErr   error
	locationCalls int
	locationLat   int32
	locationLon   int32
	locationAlt   int32

	resetErr   error
	resetCalls int
}

func (f *fakeClient) GetRadioInfo() ([]*pb.FromRadio, error) { return f.infoResponses, f.infoErr }
func (f *fakeClient) SendTextMessage(message string, to int64, channel int64) error {
	f.sendCalls++
	f.sendMessage = message
	f.sendTo = to
	f.sendChannel = channel
	return f.sendErr
}
func (f *fakeClient) SetRadioOwner(name string) error {
	f.ownerCalls++
	f.ownerName = name
	return f.ownerErr
}
func (f *fakeClient) SetModemMode(mode string) error {
	f.modemCalls++
	f.modemMode = mode
	return f.modemErr
}
func (f *fakeClient) SetLocation(lat int32, long int32, alt int32) error {
	f.locationCalls++
	f.locationLat = lat
	f.locationLon = long
	f.locationAlt = alt
	return f.locationErr
}
func (f *fakeClient) FactoryReset() error {
	f.resetCalls++
	return f.resetErr
}

func TestServiceSendTextValidationAndSuccess(t *testing.T) {
	svc := NewService(&fakeClient{})
	ctx := context.Background()

	_, err := svc.SendText(ctx, SendTextRequest{Message: " ", To: 0, Channel: 0})
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation error, got %v", err)
	}

	fc := &fakeClient{}
	svc = NewService(fc)
	res, err := svc.SendText(ctx, SendTextRequest{Message: "hi", To: 12, Channel: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Message != "hi" || res.To != 12 || res.Channel != 3 {
		t.Fatalf("unexpected response: %+v", res)
	}
	if fc.sendCalls != 1 || fc.sendMessage != "hi" || fc.sendTo != 12 || fc.sendChannel != 3 {
		t.Fatalf("unexpected send call")
	}
}

func TestServiceSetOwnerAndModem(t *testing.T) {
	ctx := context.Background()
	fc := &fakeClient{}
	svc := NewService(fc)

	ownerRes, err := svc.SetOwner(ctx, SetOwnerRequest{Name: "Moon Station"})
	if err != nil {
		t.Fatalf("unexpected owner error: %v", err)
	}
	if ownerRes.Name != "Moon Station" || fc.ownerName != "Moon Station" || fc.ownerCalls != 1 {
		t.Fatalf("unexpected owner result/call")
	}

	modemRes, err := svc.SetModem(ctx, SetModemRequest{Mode: "LF"})
	if err != nil {
		t.Fatalf("unexpected modem error: %v", err)
	}
	if modemRes.Mode != "lf" || fc.modemMode != "lf" || fc.modemCalls != 1 {
		t.Fatalf("unexpected modem result/call")
	}
}

func TestServiceSetLocationValidationAndSuccess(t *testing.T) {
	ctx := context.Background()
	fc := &fakeClient{}
	svc := NewService(fc)

	_, err := svc.SetLocation(ctx, SetLocationRequest{LatI: 1<<40 - 1, LonI: 0, Alt: 0})
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation error, got %v", err)
	}

	res, err := svc.SetLocation(ctx, SetLocationRequest{LatI: 1, LonI: -2, Alt: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.LatI != 1 || res.LonI != -2 || res.Alt != 3 {
		t.Fatalf("unexpected location response: %+v", res)
	}
	if fc.locationCalls != 1 || fc.locationLat != 1 || fc.locationLon != -2 || fc.locationAlt != 3 {
		t.Fatalf("unexpected location call")
	}
}

func TestServiceInfoAndFactoryReset(t *testing.T) {
	ctx := context.Background()
	fc := &fakeClient{
		infoResponses: []*pb.FromRadio{
			{PayloadVariant: &pb.FromRadio_MyInfo{MyInfo: &pb.MyNodeInfo{MyNodeNum: 0x16c3f424}}},
			{PayloadVariant: &pb.FromRadio_Metadata{Metadata: &pb.DeviceMetadata{FirmwareVersion: "2.6.0"}}},
			{PayloadVariant: &pb.FromRadio_NodeInfo{NodeInfo: &pb.NodeInfo{}}},
		},
	}
	svc := NewService(fc)

	info, err := svc.Info(ctx)
	if err != nil {
		t.Fatalf("unexpected info error: %v", err)
	}
	if info.Summary.Responses != 3 || info.Summary.Nodes != 1 || info.Summary.MyNode != "!16c3f424" {
		t.Fatalf("unexpected info summary: %+v", info.Summary)
	}

	if err := svc.FactoryReset(ctx); err != nil {
		t.Fatalf("unexpected factory reset error: %v", err)
	}
	if fc.resetCalls != 1 {
		t.Fatalf("reset calls = %d, want 1", fc.resetCalls)
	}
}
