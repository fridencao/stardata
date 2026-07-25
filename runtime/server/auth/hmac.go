package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/protobuf/encoding/protojson"
)

// TokenValidator validates a bearer token and returns a ClaimsProvider.
// Both *Audience (RS256/JWKS, the admin flow) and *HMACAudience (HS256, self-hosted)
// implement this interface, so the interceptors can accept either without caring about the signing algorithm.
type TokenValidator interface {
	ParseAndValidate(tokenStr string) (ClaimsProvider, error)
}

// HMACIssuer signs JWTs using HMAC-SHA256 with a shared secret.
// It is the simplest option for self-hosted deployments: no JWKS endpoint needs to be served,
// and the same secret is used for both signing and validation.
type HMACIssuer struct {
	secret    []byte
	issuerURL string
}

// NewHMACIssuer creates an issuer that signs tokens with the given secret.
func NewHMACIssuer(issuerURL, secret string) (*HMACIssuer, error) {
	if secret == "" {
		return nil, errors.New("jwt secret must not be empty")
	}
	return &HMACIssuer{secret: []byte(secret), issuerURL: issuerURL}, nil
}

// NewToken issues a new HS256 JWT based on the provided options.
func (i *HMACIssuer) NewToken(opts TokenOptions) (string, error) {
	claims := buildJWTClaims(i.issuerURL, opts)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	res, err := token.SignedString(i.secret)
	if err != nil {
		return "", err
	}
	return res, nil
}

// HMACAudience validates HS256 JWTs signed with a shared secret.
type HMACAudience struct {
	issuerURL   string
	audienceURL string
	secret       []byte
}

// NewHMACAudience creates an audience that validates HS256 tokens.
func NewHMACAudience(issuerURL, audienceURL, secret string) (*HMACAudience, error) {
	if secret == "" {
		return nil, errors.New("jwt secret must not be empty")
	}
	return &HMACAudience{issuerURL: issuerURL, audienceURL: audienceURL, secret: []byte(secret)}, nil
}

// ParseAndValidate parses and validates an HS256 JWT and returns its claims.
func (a *HMACAudience) ParseAndValidate(tokenStr string) (ClaimsProvider, error) {
	claims := &jwtClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %q", t.Header["alg"])
		}
		return a.secret, nil
	})
	if err != nil {
		return nil, err
	}

	if !claims.VerifyIssuer(a.issuerURL, true) {
		return nil, fmt.Errorf("invalid token issuer %q (expected %q)", claims.Issuer, a.issuerURL)
	}

	if !claims.VerifyAudience(a.audienceURL, true) {
		return nil, fmt.Errorf("invalid token audience %q (expected %q)", claims.Audience, a.audienceURL)
	}

	return claims, nil
}

// buildJWTClaims assembles a jwtClaims value from TokenOptions.
// Kept in one place so that HMACIssuer and the RS256 Issuer stay in sync.
func buildJWTClaims(issuerURL string, opts TokenOptions) *jwtClaims {
	var sec []json.RawMessage
	if len(opts.SecurityRules) > 0 {
		sec = make([]json.RawMessage, len(opts.SecurityRules))
		for i, rule := range opts.SecurityRules {
			data, err := protojson.Marshal(rule)
			if err != nil {
				// protojson errors are not recoverable here; the issuer should surface them.
				panic(err)
			}
			sec[i] = data
		}
	}

	now := time.Now()
	return &jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(opts.TTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    issuerURL,
			Subject:   opts.Subject,
			Audience:  []string{opts.AudienceURL},
		},
		System:    opts.SystemPermissions,
		Instances: opts.InstancePermissions,
		Attrs:     opts.Attributes,
		Security:  sec,
	}
}
