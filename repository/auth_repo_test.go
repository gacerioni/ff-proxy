package repository

import (
	"context"
	"testing"

	"github.com/harness/ff-proxy/cache"

	"github.com/harness/ff-proxy/domain"
	"github.com/stretchr/testify/assert"
)

var (
	emptyApprovedEnvironments = map[string]struct{}{}
)

func TestAuthRepo_Get(t *testing.T) {
	populated := map[domain.AuthAPIKey]string{
		domain.AuthAPIKey("apikey-foo"): "env-approved",
		domain.AuthAPIKey("apikey-2"):   "env-not-approved",
	}
	unpopulated := map[domain.AuthAPIKey]string{}

	type expected struct {
		strVal  string
		boolVal bool
	}

	testCases := map[string]struct {
		cache        cache.Cache
		data         map[domain.AuthAPIKey]string
		approvedEnvs map[string]struct{}
		key          string
		expected     expected
	}{
		"Given I have an empty AuthRepo": {
			cache:        cache.NewMemCache(),
			data:         unpopulated,
			approvedEnvs: emptyApprovedEnvironments,
			key:          "apikey-foo",
			expected:     expected{strVal: "", boolVal: false},
		},
		"Given I have a populated AuthRepo but try to get a key that doesn't exist": {
			cache:        cache.NewMemCache(),
			data:         populated,
			approvedEnvs: emptyApprovedEnvironments,
			key:          "foo",
			expected:     expected{strVal: "", boolVal: false},
		},
		"Given I have a populated AuthRepo and try to get a key that does exist": {
			cache:        cache.NewMemCache(),
			data:         populated,
			approvedEnvs: emptyApprovedEnvironments,
			key:          "apikey-foo",
			expected:     expected{strVal: "env-approved", boolVal: true},
		},
		"Given I have a populated AuthRepo and try to get a key that is on the approved env list": {
			cache:        cache.NewMemCache(),
			data:         populated,
			approvedEnvs: map[string]struct{}{"env-approved": struct{}{}},
			key:          "apikey-foo",
			expected:     expected{strVal: "env-approved", boolVal: true},
		},
		"Given I have a populated AuthRepo and try to get a key that isn't on the approved env list": {
			cache:        cache.NewMemCache(),
			data:         populated,
			approvedEnvs: map[string]struct{}{"env-approved": struct{}{}},
			key:          "apikey-2",
			expected:     expected{strVal: "", boolVal: false},
		},
	}
	for desc, tc := range testCases {
		tc := tc
		t.Run(desc, func(t *testing.T) {

			repo, err := NewAuthRepo(tc.cache, tc.data, tc.approvedEnvs)
			if err != nil {
				t.Fatalf("(%s): error = %v", desc, err)
			}
			actual, ok := repo.Get(context.Background(), domain.AuthAPIKey(tc.key))

			assert.Equal(t, tc.expected.boolVal, ok)
			assert.Equal(t, tc.expected.strVal, actual)
		})
	}
}

func TestAuthRepo_GetAll(t *testing.T) {
	populated := map[domain.AuthAPIKey]string{
		domain.AuthAPIKey("apikey-foo"): "env-foo",
		domain.AuthAPIKey("apikey-bar"): "env-bar",
	}

	unpopulated := map[domain.AuthAPIKey]string{}

	extraKeys := map[domain.AuthAPIKey]string{
		domain.AuthAPIKey("apikey-extra"): "env-extra",
	}

	type expected struct {
		keys    map[domain.AuthAPIKey]string
		boolVal bool
	}

	testCases := map[string]struct {
		cache        cache.Cache
		data         map[domain.AuthAPIKey]string
		approvedEnvs map[string]struct{}
		fn           func(repo AuthRepo)
		key          string
		expected     expected
	}{
		"Given I have an empty AuthRepo": {
			cache:    cache.NewMemCache(),
			data:     unpopulated,
			expected: expected{keys: map[domain.AuthAPIKey]string{}, boolVal: false},
		},
		"Given I have a populated AuthRepo": {
			cache:    cache.NewMemCache(),
			data:     populated,
			expected: expected{keys: populated, boolVal: true},
		},
		"Given I have a populated AuthRepo and approved env list with all envs": {
			cache:        cache.NewMemCache(),
			data:         populated,
			approvedEnvs: map[string]struct{}{"env-foo": struct{}{}, "env-bar": struct{}{}},
			expected:     expected{keys: populated, boolVal: true},
		},
		"Given I have a populated AuthRepo and approved env list with one env": {
			cache:        cache.NewMemCache(),
			data:         populated,
			approvedEnvs: map[string]struct{}{"env-foo": struct{}{}},
			expected: expected{keys: map[domain.AuthAPIKey]string{
				domain.AuthAPIKey("apikey-foo"): "env-foo",
			}, boolVal: true},
		},
		"Given I have a populated AuthRepo and approved env list with env with no results": {
			cache:        cache.NewMemCache(),
			data:         populated,
			approvedEnvs: map[string]struct{}{"env-noexist": struct{}{}},
			expected:     expected{keys: map[domain.AuthAPIKey]string{}, boolVal: false},
		},
		"Given I add to the  AuthRepo": {
			cache: cache.NewMemCache(),
			data:  populated,
			fn: func(repo AuthRepo) {
				for key, env := range extraKeys {
					repo.Add(context.Background(), domain.AuthConfig{
						APIKey:        key,
						EnvironmentID: domain.EnvironmentID(env),
					})
				}

			},
			expected: expected{keys: mergeAuthMaps(populated, extraKeys), boolVal: true},
		},
	}
	for desc, tc := range testCases {
		tc := tc
		t.Run(desc, func(t *testing.T) {

			repo, err := NewAuthRepo(tc.cache, tc.data, tc.approvedEnvs)
			if err != nil {
				t.Fatalf("(%s): error = %v", desc, err)
			}
			if tc.fn != nil {
				tc.fn(repo)
			}
			actual, ok := repo.getAll(context.Background())

			assert.Equal(t, tc.expected.boolVal, ok)
			assert.Equal(t, tc.expected.keys, actual)
		})
	}
}

func TestAuthRepo_Setup(t *testing.T) {
	// we start with two keys for the foo environment
	fooKeys := map[domain.AuthAPIKey]string{
		domain.AuthAPIKey("apikey-foo"):  "env-foo",
		domain.AuthAPIKey("apikey-foo2"): "env-foo",
	}

	// we test adding new auth data for a different env to make sure we don't clear data from other envs
	barKeys := map[domain.AuthAPIKey]string{
		domain.AuthAPIKey("apikey-bar"): "env-bar",
	}

	// we also test adding fresh data for foo env to make sure we clear old keys
	newFooKeys := map[domain.AuthAPIKey]string{
		domain.AuthAPIKey("apikey-foo"): "env-foo",
	}

	type expected struct {
		keys map[domain.AuthAPIKey]string
	}

	testCases := map[string]struct {
		initialData map[domain.AuthAPIKey]string
		extraData   map[domain.AuthAPIKey]string
		cache       cache.Cache
		data        map[domain.AuthAPIKey]string
		key         string
		expected    expected
	}{
		"Given I add initial data to empty cache": {
			cache:       cache.NewMemCache(),
			initialData: fooKeys,

			expected: expected{keys: fooKeys},
		},
		"Given I add extra env keys to populated cache": {
			cache:       cache.NewMemCache(),
			initialData: fooKeys,
			extraData:   barKeys,
			expected:    expected{keys: mergeAuthMaps(fooKeys, barKeys)},
		},
		"Given I alter env keys for populated cache": {
			cache:       cache.NewMemCache(),
			initialData: fooKeys,
			extraData:   newFooKeys,
			expected:    expected{keys: newFooKeys},
		},
	}
	for desc, tc := range testCases {
		tc := tc
		t.Run(desc, func(t *testing.T) {

			// populate initial data
			_, err := NewAuthRepo(tc.cache, tc.initialData, emptyApprovedEnvironments)
			if err != nil {
				t.Fatalf("(%s): error = %v", desc, err)
			}

			// populate extra data
			repo, err := NewAuthRepo(tc.cache, tc.extraData, emptyApprovedEnvironments)
			if err != nil {
				t.Fatalf("(%s): error = %v", desc, err)
			}

			actual, _ := repo.getAll(context.Background())

			assert.Equal(t, tc.expected.keys, actual)
		})
	}
}

// merge any number of auth maps into one
// used to produce expected test results easier
func mergeAuthMaps(maps ...map[domain.AuthAPIKey]string) map[domain.AuthAPIKey]string {
	newMap := map[domain.AuthAPIKey]string{}
	for _, m := range maps {
		for k, v := range m {
			newMap[k] = v
		}
	}

	return newMap
}
