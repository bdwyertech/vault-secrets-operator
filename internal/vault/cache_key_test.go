// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vault

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	secretsv1beta1 "github.com/hashicorp/vault-secrets-operator/api/v1beta1"
	"github.com/hashicorp/vault-secrets-operator/internal/vault/credentials"
)

const (
	authUID      = types.UID("c4fad6b9-e7bb-4ed8-bc38-67fd6dc85a35")
	connUID      = types.UID("c4fad6b9-e7bb-4ed8-bc38-67fd6dc85a36")
	providerUID  = types.UID("c4fad6b9-e7bb-4ed8-bc38-67fd6dc85a37")
	computedHash = "2a8108711ae49ac0faa724"
)

type computeClientCacheKeyTest struct {
	name        string
	authObj     *secretsv1beta1.VaultAuth
	connObj     *secretsv1beta1.VaultConnection
	providerUID types.UID
	want        ClientCacheKey
	wantErr     assert.ErrorAssertionFunc
}

func Test_computeClientCacheKey(t *testing.T) {
	type args struct{}
	tests := []computeClientCacheKeyTest{
		{
			name: "valid",
			authObj: &secretsv1beta1.VaultAuth{
				ObjectMeta: metav1.ObjectMeta{
					UID:        authUID,
					Generation: 0,
				},
				Spec: secretsv1beta1.VaultAuthSpec{
					Method: "ical",
				},
			},
			connObj: &secretsv1beta1.VaultConnection{
				ObjectMeta: metav1.ObjectMeta{
					UID:        connUID,
					Generation: 0,
				},
			},
			providerUID: providerUID,
			want:        "ical-" + computedHash,
			wantErr:     assert.NoError,
		},
		{
			name: "valid-key-at-max-length",
			authObj: &secretsv1beta1.VaultAuth{
				ObjectMeta: metav1.ObjectMeta{
					UID:        authUID,
					Generation: 0,
				},
				Spec: secretsv1beta1.VaultAuthSpec{
					Method: "ical" + strings.Repeat("x", 36),
				},
			},
			connObj: &secretsv1beta1.VaultConnection{
				ObjectMeta: metav1.ObjectMeta{
					UID:        connUID,
					Generation: 0,
				},
			},
			providerUID: providerUID,
			want:        ClientCacheKey("ical" + strings.Repeat("x", 36) + "-" + computedHash),
			wantErr:     assert.NoError,
		},
		{
			name: "invalid-key-max-length-exceeded",
			authObj: &secretsv1beta1.VaultAuth{
				ObjectMeta: metav1.ObjectMeta{
					UID:        authUID,
					Generation: 0,
				},
				Spec: secretsv1beta1.VaultAuthSpec{
					Method: "ical" + strings.Repeat("x", 37),
				},
			},
			connObj: &secretsv1beta1.VaultConnection{
				ObjectMeta: metav1.ObjectMeta{
					UID:        connUID,
					Generation: 0,
				},
			},
			providerUID: providerUID,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return !assert.ErrorIs(t, err, errorKeyLengthExceeded, i...)
			},
		},
		{
			name: "invalid-duplicate-uid",
			authObj: &secretsv1beta1.VaultAuth{
				ObjectMeta: metav1.ObjectMeta{
					UID:        authUID,
					Generation: 0,
				},
				Spec: secretsv1beta1.VaultAuthSpec{
					Method: "ical",
				},
			},
			connObj: &secretsv1beta1.VaultConnection{
				ObjectMeta: metav1.ObjectMeta{
					UID:        authUID,
					Generation: 0,
				},
			},
			providerUID: providerUID,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return !assert.ErrorIs(t, err, errorDuplicateUID, i...)
			},
		},
		{
			name: "invalid-uid-length-above",
			authObj: &secretsv1beta1.VaultAuth{
				ObjectMeta: metav1.ObjectMeta{
					UID:        authUID + "1",
					Generation: 0,
				},
				Spec: secretsv1beta1.VaultAuthSpec{
					Method: "ical",
				},
			},
			connObj: &secretsv1beta1.VaultConnection{
				ObjectMeta: metav1.ObjectMeta{
					UID:        authUID,
					Generation: 0,
				},
			},
			providerUID: providerUID,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return !assert.ErrorIs(t, err, errorInvalidUIDLength, i...)
			},
		},
		{
			name: "invalid-uid-length-below",
			authObj: &secretsv1beta1.VaultAuth{
				ObjectMeta: metav1.ObjectMeta{
					UID:        authUID[0 : len(authUID)-1],
					Generation: 0,
				},
				Spec: secretsv1beta1.VaultAuthSpec{
					Method: "ical",
				},
			},
			connObj: &secretsv1beta1.VaultConnection{
				ObjectMeta: metav1.ObjectMeta{
					UID:        authUID,
					Generation: 0,
				},
			},
			providerUID: providerUID,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return !assert.ErrorIs(t, err, errorInvalidUIDLength, i...)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := computeClientCacheKey(tt.authObj, tt.connObj, tt.providerUID)
			if !tt.wantErr(t, err, fmt.Sprintf("computeClientCacheKey(%v, %v, %v)",
				tt.authObj, tt.connObj, tt.providerUID)) {
				return
			}
			assert.Equalf(t, tt.want, got, "computeClientCacheKey(%v, %v, %v)", tt.authObj, tt.connObj, tt.providerUID)
		})
	}
}

func TestComputeClientCacheKeyFromClient(t *testing.T) {
	tests := []computeClientCacheKeyTest{
		{
			name: "valid",
			authObj: &secretsv1beta1.VaultAuth{
				ObjectMeta: metav1.ObjectMeta{
					UID:        authUID,
					Generation: 0,
				},
				Spec: secretsv1beta1.VaultAuthSpec{
					Method: "ical",
				},
			},
			connObj: &secretsv1beta1.VaultConnection{
				ObjectMeta: metav1.ObjectMeta{
					UID:        connUID,
					Generation: 0,
				},
			},
			providerUID: providerUID,
			want:        ClientCacheKey("ical-" + computedHash),
			wantErr:     assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Client
			if tt.authObj == nil || tt.connObj == nil || tt.providerUID == "" {
				c = &defaultClient{}
			} else {
				c = &defaultClient{
					authObj: tt.authObj,
					connObj: tt.connObj,
					credentialProvider: credentials.NewKubernetesCredentialProvider(nil, "",
						tt.providerUID),
				}
			}

			got, err := ComputeClientCacheKeyFromClient(c)
			if !tt.wantErr(t, err, fmt.Sprintf("ComputeClientCacheKeyFromClient(%v)", c)) {
				return
			}
			assert.Equalf(t, tt.want, got, "ComputeClientCacheKeyFromClient(%v)", c)
		})
	}
}

func TestClientCacheKey_IsClone(t *testing.T) {
	tests := []struct {
		name string
		k    ClientCacheKey
		want bool
	}{
		{
			name: "is-not-a-clone-no-suffix",
			k: ClientCacheKey(fmt.Sprintf("%s-%s",
				credentials.ProviderMethodKubernetes,
				computedHash)),
			want: false,
		},
		{
			name: "is-not-a-clone-empty-suffix",
			k: ClientCacheKey(fmt.Sprintf("%s-%s-",
				credentials.ProviderMethodKubernetes,
				computedHash)),
			want: false,
		},
		{
			name: "is-a-clone",
			k: ClientCacheKey(fmt.Sprintf("%s-%s-ns1/ns2",
				credentials.ProviderMethodKubernetes,
				computedHash)),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.k.IsClone(), "IsClone()")
		})
	}
}

func TestClientCacheKeyClone(t *testing.T) {
	tests := []struct {
		name      string
		key       ClientCacheKey
		namespace string
		want      ClientCacheKey
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name: "valid",
			key: ClientCacheKey(fmt.Sprintf("%s-%s",
				credentials.ProviderMethodKubernetes,
				computedHash)),
			namespace: "ns1/ns2",
			want: ClientCacheKey(fmt.Sprintf("%s-%s-ns1/ns2",
				credentials.ProviderMethodKubernetes,
				computedHash)),
			wantErr: assert.NoError,
		},
		{
			name: "fail-empty-namespace",
			key: ClientCacheKey(fmt.Sprintf("%s-%s",
				credentials.ProviderMethodKubernetes,
				computedHash)),
			namespace: "",
			want:      "",
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.EqualError(t, err, "namespace cannot be empty")
			},
		},
		{
			name: "fail-parent-is-clone",
			key: ClientCacheKey(fmt.Sprintf("%s-%s-ns1/ns2",
				credentials.ProviderMethodKubernetes,
				computedHash)),
			namespace: "ns3",
			want:      "",
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.EqualError(t, err, "parent key cannot be a clone")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ClientCacheKeyClone(tt.key, tt.namespace)
			if !tt.wantErr(t, err, fmt.Sprintf("ClientCacheKeyClone(%v, %v)", tt.key, tt.namespace)) {
				return
			}
			assert.Equalf(t, tt.want, got, "ClientCacheKeyClone(%v, %v)", tt.key, tt.namespace)
		})
	}
}
