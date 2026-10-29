package llm

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/big"
	"regexp"
	"strings"

	"Q-Solver/pkg/llm/protocol"
)

type ProbeKind string

const (
	ProbeText   ProbeKind = "text"
	ProbeVision ProbeKind = "vision"
)

type ProbeOutcome string

const (
	ProbePassed       ProbeOutcome = "passed"
	ProbeFailed       ProbeOutcome = "failed"
	ProbeInconclusive ProbeOutcome = "inconclusive"
)

type ProbeRequest struct {
	Kind string `json:"kind"`
}

type ProbeResult struct {
	Kind        string               `json:"kind"`
	Outcome     ProbeOutcome         `json:"outcome"`
	Protocol    protocol.APIProtocol `json:"protocol"`
	Message     string               `json:"message"`
	Error       *ProviderError       `json:"error,omitempty"`
	RequestSent bool                 `json:"requestSent"`
}

type Prober interface {
	Probe(context.Context, ProbeRequest) ProbeResult
}

type providerProber struct {
	provider Provider
	protocol protocol.APIProtocol
	model    string
}

func (p providerProber) Probe(ctx context.Context, request ProbeRequest) ProbeResult {
	return ProbeProvider(ctx, p.provider, p.protocol, p.model, request)
}

type textProbeGenerator interface {
	generateTextProbe(context.Context, string, []Message) (Message, error)
}

func ProbeProvider(ctx context.Context, provider Provider, p protocol.APIProtocol, model string, request ProbeRequest) ProbeResult {
	switch ProbeKind(strings.TrimSpace(request.Kind)) {
	case ProbeText:
		token, err := newProbeToken()
		if err != nil {
			return failedProbe(string(ProbeText), p, NewProviderError(ErrorInvalidRequest, StageValidateProbe, p, err), false)
		}
		return probeTextWithToken(ctx, provider, p, model, token)
	case ProbeVision:
		digits, imageData, err := newVisionChallenge()
		if err != nil {
			return failedProbe(string(ProbeVision), p, NewProviderError(ErrorInvalidRequest, StageValidateProbe, p, err), false)
		}
		return probeVisionWithChallenge(ctx, provider, p, model, digits, imageData)
	default:
		return failedProbe(request.Kind, p, NewProviderError(ErrorInvalidRequest, StageValidateProbe, p, nil), false)
	}
}

func probeTextWithToken(ctx context.Context, provider Provider, p protocol.APIProtocol, model, token string) ProbeResult {
	prompt := "Return exactly this verification token and nothing else: " + token
	messages := []Message{NewUserMessage(prompt)}
	var response Message
	var err error
	if generator, ok := provider.(textProbeGenerator); ok {
		response, err = generator.generateTextProbe(ctx, model, messages)
	} else {
		response, err = provider.GenerateContent(ctx, model, messages)
	}
	if err != nil {
		return failedProbe(string(ProbeText), p, normalizeProbeError(err, p), true)
	}
	if strings.TrimSpace(response.Content) != token {
		return failedProbe(string(ProbeText), p, NewProviderError(ErrorInvalidResponse, StageValidateProbe, p, nil), true)
	}
	return ProbeResult{Kind: string(ProbeText), Outcome: ProbePassed, Protocol: p, Message: "文字响应验证通过", RequestSent: true}
}

func probeVisionWithChallenge(ctx context.Context, provider Provider, p protocol.APIProtocol, model, digits string, imageData []byte) ProbeResult {
	prompt := "Read the two digits in this synthetic test image. Reply exactly as DIGITS: NN, replacing NN with only the two digits you see."
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageData)
	messages := []Message{NewMultiPartMessage(RoleUser, []ContentPart{TextPart(prompt), ImagePart(dataURL)})}
	response, err := provider.GenerateContentStream(ctx, messages, nil)
	if err != nil {
		return failedProbe(string(ProbeVision), p, normalizeProbeError(err, p), true)
	}
	got, ok := normalizeVisionAnswer(response.Content)
	if !ok {
		probeErr := NewProviderError(ErrorVisionInconclusive, StageValidateProbe, p, nil)
		return ProbeResult{Kind: string(ProbeVision), Outcome: ProbeInconclusive, Protocol: p, Message: probeErr.Message, Error: probeErr, RequestSent: true}
	}
	if got != digits {
		return failedProbe(string(ProbeVision), p, NewProviderError(ErrorVisionMismatch, StageValidateProbe, p, nil), true)
	}
	return ProbeResult{Kind: string(ProbeVision), Outcome: ProbePassed, Protocol: p, Message: "图片识别验证通过", RequestSent: true}
}

func failedProbe(kind string, p protocol.APIProtocol, err *ProviderError, sent bool) ProbeResult {
	return ProbeResult{Kind: kind, Outcome: ProbeFailed, Protocol: p, Message: err.Message, Error: err, RequestSent: sent}
}

func normalizeProbeError(err error, p protocol.APIProtocol) *ProviderError {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr
	}
	return ClassifyProviderError(err, StageValidateProbe, p)
}

func newProbeToken() (string, error) {
	var data [6]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return "QSP-" + strings.ToUpper(hex.EncodeToString(data[:])), nil
}

func newVisionChallenge() (string, []byte, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		return "", nil, err
	}
	digits := fmt.Sprintf("%02d", value.Int64())
	data, err := renderSevenSegmentPNG(digits)
	return digits, data, err
}

var visionAnswerPattern = regexp.MustCompile(`^(?:DIGITS:\s*)?([0-9]{2})$`)

func normalizeVisionAnswer(value string) (string, bool) {
	value = strings.TrimSpace(value)
	var normalized strings.Builder
	for _, r := range value {
		if r >= '０' && r <= '９' {
			r = '0' + (r - '０')
		}
		normalized.WriteRune(r)
	}
	matches := visionAnswerPattern.FindStringSubmatch(normalized.String())
	if len(matches) != 2 {
		return "", false
	}
	return matches[1], true
}

var sevenSegmentDigits = [10][7]bool{
	{true, true, true, true, true, true, false},
	{false, true, true, false, false, false, false},
	{true, true, false, true, true, false, true},
	{true, true, true, true, false, false, true},
	{false, true, true, false, false, true, true},
	{true, false, true, true, false, true, true},
	{true, false, true, true, true, true, true},
	{true, true, true, false, false, false, false},
	{true, true, true, true, true, true, true},
	{true, true, true, true, false, true, true},
}

func renderSevenSegmentPNG(digits string) ([]byte, error) {
	if len(digits) != 2 || digits[0] < '0' || digits[0] > '9' || digits[1] < '0' || digits[1] > '9' {
		return nil, fmt.Errorf("invalid seven-segment challenge")
	}
	canvas := image.NewRGBA(image.Rect(0, 0, 140, 84))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	for index := 0; index < 2; index++ {
		drawSevenSegmentDigit(canvas, 12+index*66, 7, int(digits[index]-'0'))
	}
	var output bytes.Buffer
	if err := png.Encode(&output, canvas); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func drawSevenSegmentDigit(canvas *image.RGBA, x, y, digit int) {
	segments := []image.Rectangle{
		image.Rect(x+8, y, x+42, y+8),
		image.Rect(x+42, y+6, x+50, y+36),
		image.Rect(x+42, y+38, x+50, y+68),
		image.Rect(x+8, y+66, x+42, y+74),
		image.Rect(x, y+38, x+8, y+68),
		image.Rect(x, y+6, x+8, y+36),
		image.Rect(x+8, y+33, x+42, y+41),
	}
	black := &image.Uniform{C: color.Black}
	for i, enabled := range sevenSegmentDigits[digit] {
		if enabled {
			draw.Draw(canvas, segments[i], black, image.Point{}, draw.Src)
		}
	}
}
