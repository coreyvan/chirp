package node

import (
	"context"
	"fmt"
	"math"
	"strings"

	pb "github.com/coreyvan/chirp/protogen/meshtastic"
)

type Client interface {
	GetRadioInfo() ([]*pb.FromRadio, error)
	SendTextMessage(message string, to int64, channel int64) error
	SetRadioOwner(name string) error
	SetModemMode(mode string) error
	SetLocation(lat int32, long int32, alt int32) error
	FactoryReset() error
}

type Service struct {
	client Client
}

func NewService(client Client) *Service {
	return &Service{client: client}
}

type InfoSummary struct {
	Responses     int    `json:"responses"`
	MyNode        string `json:"my_node"`
	Firmware      string `json:"firmware"`
	HWModel       string `json:"hw_model"`
	Role          string `json:"role"`
	Nodes         int    `json:"nodes"`
	Channels      int    `json:"channels"`
	Configs       int    `json:"configs"`
	ModuleConfigs int    `json:"module_configs"`
}

type InfoResult struct {
	Summary   InfoSummary
	Responses []*pb.FromRadio
}

func (s *Service) Info(_ context.Context) (InfoResult, error) {
	responses, err := s.client.GetRadioInfo()
	if err != nil {
		return InfoResult{}, fmt.Errorf("get radio info: %w", err)
	}

	return InfoResult{
		Summary:   BuildInfoSummary(responses),
		Responses: responses,
	}, nil
}

func BuildInfoSummary(responses []*pb.FromRadio) InfoSummary {
	summary := InfoSummary{
		Responses: len(responses),
		MyNode:    "-",
		Firmware:  "-",
		HWModel:   "-",
		Role:      "-",
	}

	for _, fr := range responses {
		switch v := fr.GetPayloadVariant().(type) {
		case *pb.FromRadio_MyInfo:
			if v.MyInfo != nil {
				summary.MyNode = fmt.Sprintf("!%08x", v.MyInfo.GetMyNodeNum())
			}
		case *pb.FromRadio_Metadata:
			if v.Metadata != nil {
				if fw := v.Metadata.GetFirmwareVersion(); fw != "" {
					summary.Firmware = fw
				}
				summary.HWModel = v.Metadata.GetHwModel().String()
				summary.Role = v.Metadata.GetRole().String()
			}
		case *pb.FromRadio_NodeInfo:
			summary.Nodes++
		case *pb.FromRadio_Channel:
			summary.Channels++
		case *pb.FromRadio_Config:
			summary.Configs++
		case *pb.FromRadio_ModuleConfig:
			summary.ModuleConfigs++
		}
	}

	return summary
}

type SendTextRequest struct {
	Message string
	To      int64
	Channel int64
}

type SendTextResult struct {
	Message string `json:"message"`
	To      int64  `json:"to"`
	Channel int64  `json:"channel"`
}

func ValidateSendTextRequest(req SendTextRequest) error {
	if strings.TrimSpace(req.Message) == "" {
		return invalidf("--message cannot be empty")
	}
	if req.To < 0 {
		return invalidf("--to must be >= 0")
	}
	if req.Channel < 0 {
		return invalidf("--channel must be >= 0")
	}
	return nil
}

func (s *Service) SendText(_ context.Context, req SendTextRequest) (SendTextResult, error) {
	if err := ValidateSendTextRequest(req); err != nil {
		return SendTextResult{}, err
	}

	if err := s.client.SendTextMessage(req.Message, req.To, req.Channel); err != nil {
		return SendTextResult{}, fmt.Errorf("send text: %w", err)
	}

	return SendTextResult{
		Message: req.Message,
		To:      req.To,
		Channel: req.Channel,
	}, nil
}

type SetOwnerRequest struct {
	Name string
}

type SetOwnerResult struct {
	Name string `json:"name"`
}

func ValidateSetOwnerRequest(req SetOwnerRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return invalidf("--name cannot be empty")
	}
	return nil
}

func (s *Service) SetOwner(_ context.Context, req SetOwnerRequest) (SetOwnerResult, error) {
	if err := ValidateSetOwnerRequest(req); err != nil {
		return SetOwnerResult{}, err
	}
	if err := s.client.SetRadioOwner(req.Name); err != nil {
		return SetOwnerResult{}, fmt.Errorf("set owner: %w", err)
	}
	return SetOwnerResult{Name: req.Name}, nil
}

var validModemModes = map[string]struct{}{
	"lf":  {},
	"ls":  {},
	"vls": {},
	"ms":  {},
	"mf":  {},
	"sl":  {},
	"sf":  {},
	"lm":  {},
}

type SetModemRequest struct {
	Mode string
}

type SetModemResult struct {
	Mode string `json:"mode"`
}

func NormalizeAndValidateModemMode(mode string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	if normalized == "" {
		return "", invalidf("--mode cannot be empty")
	}
	if _, ok := validModemModes[normalized]; !ok {
		return "", invalidf("--mode must be one of: lf|ls|vls|ms|mf|sl|sf|lm")
	}
	return normalized, nil
}

func (s *Service) SetModem(_ context.Context, req SetModemRequest) (SetModemResult, error) {
	mode, err := NormalizeAndValidateModemMode(req.Mode)
	if err != nil {
		return SetModemResult{}, err
	}
	if err := s.client.SetModemMode(mode); err != nil {
		return SetModemResult{}, fmt.Errorf("set modem: %w", err)
	}
	return SetModemResult{Mode: mode}, nil
}

type SetLocationRequest struct {
	LatI int64
	LonI int64
	Alt  int64
}

type SetLocationResult struct {
	LatI int32 `json:"lat_i"`
	LonI int32 `json:"lon_i"`
	Alt  int32 `json:"alt"`
}

func ValidateAndConvertSetLocationRequest(req SetLocationRequest) (SetLocationResult, error) {
	lat, err := int32FromInt64("--lat-i", req.LatI)
	if err != nil {
		return SetLocationResult{}, err
	}
	lon, err := int32FromInt64("--lon-i", req.LonI)
	if err != nil {
		return SetLocationResult{}, err
	}
	alt, err := int32FromInt64("--alt", req.Alt)
	if err != nil {
		return SetLocationResult{}, err
	}

	return SetLocationResult{
		LatI: lat,
		LonI: lon,
		Alt:  alt,
	}, nil
}

func (s *Service) SetLocation(_ context.Context, req SetLocationRequest) (SetLocationResult, error) {
	converted, err := ValidateAndConvertSetLocationRequest(req)
	if err != nil {
		return SetLocationResult{}, err
	}

	if err := s.client.SetLocation(converted.LatI, converted.LonI, converted.Alt); err != nil {
		return SetLocationResult{}, fmt.Errorf("set location: %w", err)
	}
	return converted, nil
}

func int32FromInt64(flagName string, value int64) (int32, error) {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return 0, invalidf("%s must be in int32 range", flagName)
	}
	return int32(value), nil
}

func (s *Service) FactoryReset(_ context.Context) error {
	if err := s.client.FactoryReset(); err != nil {
		return fmt.Errorf("factory-reset: %w", err)
	}
	return nil
}
