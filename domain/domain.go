package domain

import (
	"encoding/json"
	"fmt"

	"github.com/dgrijalva/jwt-go"
	admingen "github.com/harness/ff-proxy/gen/admin"
	clientgen "github.com/harness/ff-proxy/gen/client"
)

// FeatureFlagKey is the key that maps to a FeatureConfig
type FeatureFlagKey string

// NewFeatureConfigKey creates a FeatureFlagKey from an environment
func NewFeatureConfigKey(envID string) FeatureFlagKey {
	return FeatureFlagKey(fmt.Sprintf("env-%s-feature-config", envID))
}

// FeatureFlag stores feature flag data
type FeatureFlag struct {
	clientgen.FeatureConfig
}

// FeatureConfig is the type containing FeatureConfig information and is what
// we return from /GET client/env/<env>/feature-configs
type FeatureConfig struct {
	FeatureFlag
	Segments map[string]Segment `json:"segments"`
}

// MarshalBinary marshals a FeatureFlag to bytes. Currently it just uses json
// marshaling but if we want to optimise storage space we could use something
// more efficient
func (f *FeatureFlag) MarshalBinary() ([]byte, error) {
	return json.Marshal(f)
}

// UnmarshalBinary unmarshals bytes to a FeatureFlag
func (f *FeatureFlag) UnmarshalBinary(b []byte) error {
	return json.Unmarshal(b, f)
}

// TargetKey is the key that maps to a Target
type TargetKey string

// NewTargetKey creates a TargetKey from an environment
func NewTargetKey(envID string) TargetKey {
	return TargetKey(fmt.Sprintf("env-%s-target-config", envID))
}

// Target is a admingen.Target that we can declare methods on
type Target struct {
	admingen.Target
}

// MarshalBinary marshals a Target to bytes. Currently it uses json marshaling
// but if we want to optimise storage space we could use something more efficient
func (t *Target) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}

// UnmarshalBinary unmarshals bytes to a Target
func (t *Target) UnmarshalBinary(b []byte) error {
	return json.Unmarshal(b, t)
}

// SegmentKey is the key that maps to a Segment
type SegmentKey string

// NewSegmentKey creates a SegmentKey from an environment
func NewSegmentKey(envID string) SegmentKey {
	return SegmentKey(fmt.Sprintf("env-%s-segment", envID))
}

// Segment is a clientgen.Segment that we can declare methods on
type Segment struct {
	clientgen.Segment
}

// MarshalBinary marshals a Segment to bytes. Currently it uses json marshaling
// but if we want to optimise storage space we could use something more efficient
func (s *Segment) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

// UnmarshalBinary unmarshals bytes to a Segment
func (s *Segment) UnmarshalBinary(b []byte) error {
	return json.Unmarshal(b, s)
}

// AuthAPIKey is the APIKey type used for authentication lookups
type AuthAPIKey string

// Claims are custom jwt claims used by the proxy for generating a jwt token
type Claims struct {
	Environment string `json:"environment"`
	jwt.StandardClaims
}
