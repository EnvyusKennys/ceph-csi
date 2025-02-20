/*
Copyright 2020 The Ceph-CSI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	t.Parallel()
	kms := VaultTokensKMS{}

	config := make(map[string]interface{})

	// empty config map
	err := kms.parseConfig(config)
	if !errors.Is(err, errConfigOptionMissing) {
		t.Errorf("unexpected error (%T): %s", err, err)
	}

	// fill default options (normally done in initVaultTokensKMS)
	config["vaultAddress"] = "https://vault.default.cluster.svc"
	config["tenantConfigName"] = vaultTokensDefaultConfigName
	config["tenantTokenName"] = vaultTokensDefaultTokenName

	// parsing with all required options
	err = kms.parseConfig(config)
	switch {
	case err != nil:
		t.Errorf("unexpected error: %s", err)
	case kms.ConfigName != vaultTokensDefaultConfigName:
		t.Errorf("ConfigName contains unexpected value: %s", kms.ConfigName)
	case kms.TokenName != vaultTokensDefaultTokenName:
		t.Errorf("TokenName contains unexpected value: %s", kms.TokenName)
	}

	// tenant "bob" uses a different kms.ConfigName
	bob := make(map[string]interface{})
	bob["tenantConfigName"] = "the-config-from-bob"
	err = kms.parseConfig(bob)
	switch {
	case err != nil:
		t.Errorf("unexpected error: %s", err)
	case kms.ConfigName != "the-config-from-bob":
		t.Errorf("ConfigName contains unexpected value: %s", kms.ConfigName)
	}
}

// TestInitVaultTokensKMS verifies that passing partial and complex
// configurations get applied correctly.
//
// When vault.New() is called at the end of initVaultTokensKMS(), errors will
// mention the missing VAULT_TOKEN, and that is expected.
func TestInitVaultTokensKMS(t *testing.T) {
	t.Parallel()
	if true {
		// FIXME: testing only works when KUBE_CONFIG is set to a
		// cluster that has a working Vault deployment
		return
	}

	args := KMSInitializerArgs{
		Tenant:  "bob",
		Config:  make(map[string]interface{}),
		Secrets: nil,
	}

	// empty config map
	_, err := initVaultTokensKMS(args)
	if !errors.Is(err, errConfigOptionMissing) {
		t.Errorf("unexpected error (%T): %s", err, err)
	}

	// fill required options
	args.Config["vaultAddress"] = "https://vault.default.cluster.svc"

	// parsing with all required options
	_, err = initVaultTokensKMS(args)
	if err != nil && !strings.Contains(err.Error(), "VAULT_TOKEN") {
		t.Errorf("unexpected error: %s", err)
	}

	// fill tenants
	tenants := make(map[string]interface{})
	args.Config["tenants"] = tenants

	// empty tenants list
	_, err = initVaultTokensKMS(args)
	if err != nil && !strings.Contains(err.Error(), "VAULT_TOKEN") {
		t.Errorf("unexpected error: %s", err)
	}

	// add tenant "bob"
	bob := make(map[string]interface{})
	bob["vaultAddress"] = "https://vault.bob.example.org"
	args.Config["tenants"].(map[string]interface{})["bob"] = bob

	_, err = initVaultTokensKMS(args)
	if err != nil && !strings.Contains(err.Error(), "VAULT_TOKEN") {
		t.Errorf("unexpected error: %s", err)
	}
}

// TestStdVaultToCSIConfig converts a JSON document with standard VAULT_*
// environment variables to a vaultTokenConf structure.
func TestStdVaultToCSIConfig(t *testing.T) {
	t.Parallel()
	vaultConfigMap := `{
		"KMS_PROVIDER":"vaulttokens",
		"VAULT_ADDR":"https://vault.example.com",
		"VAULT_BACKEND_PATH":"/secret",
		"VAULT_CACERT":"",
		"VAULT_TLS_SERVER_NAME":"vault.example.com",
		"VAULT_CLIENT_CERT":"",
		"VAULT_CLIENT_KEY":"",
		"VAULT_NAMESPACE":"a-department",
		"VAULT_SKIP_VERIFY":"true"
	}`

	sv := &standardVault{}
	err := json.Unmarshal([]byte(vaultConfigMap), sv)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	v := vaultTokenConf{}
	v.convertStdVaultToCSIConfig(sv)

	switch {
	case v.EncryptionKMSType != kmsTypeVaultTokens:
		t.Errorf("unexpected value for EncryptionKMSType: %s", v.EncryptionKMSType)
	case v.VaultAddress != "https://vault.example.com":
		t.Errorf("unexpected value for VaultAddress: %s", v.VaultAddress)
	case v.VaultBackendPath != "/secret":
		t.Errorf("unexpected value for VaultBackendPath: %s", v.VaultBackendPath)
	case v.VaultCAFromSecret != "":
		t.Errorf("unexpected value for VaultCAFromSecret: %s", v.VaultCAFromSecret)
	case v.VaultClientCertFromSecret != "":
		t.Errorf("unexpected value for VaultClientCertFromSecret: %s", v.VaultClientCertFromSecret)
	case v.VaultClientCertKeyFromSecret != "":
		t.Errorf("unexpected value for VaultClientCertKeyFromSecret: %s", v.VaultClientCertKeyFromSecret)
	case v.VaultNamespace != "a-department":
		t.Errorf("unexpected value for VaultNamespace: %s", v.VaultNamespace)
	case v.VaultTLSServerName != "vault.example.com":
		t.Errorf("unexpected value for VaultTLSServerName: %s", v.VaultTLSServerName)
	case v.VaultCAVerify != "false":
		t.Errorf("unexpected value for VaultCAVerify: %s", v.VaultCAVerify)
	}
}

func TestTransformConfig(t *testing.T) {
	t.Parallel()
	cm := make(map[string]interface{})
	cm["KMS_PROVIDER"] = "vaulttokens"
	cm["VAULT_ADDR"] = "https://vault.example.com"
	cm["VAULT_BACKEND_PATH"] = "/secret"
	cm["VAULT_CACERT"] = ""
	cm["VAULT_TLS_SERVER_NAME"] = "vault.example.com"
	cm["VAULT_CLIENT_CERT"] = ""
	cm["VAULT_CLIENT_KEY"] = ""
	cm["VAULT_NAMESPACE"] = "a-department"
	cm["VAULT_SKIP_VERIFY"] = "true" // inverse of "vaultCAVerify"

	config, err := transformConfig(cm)
	require.NoError(t, err)
	assert.Equal(t, config["encryptionKMSType"], cm["KMS_PROVIDER"])
	assert.Equal(t, config["vaultAddress"], cm["VAULT_ADDR"])
	assert.Equal(t, config["vaultBackendPath"], cm["VAULT_BACKEND_PATH"])
	assert.Equal(t, config["vaultCAFromSecret"], cm["VAULT_CACERT"])
	assert.Equal(t, config["vaultTLSServerName"], cm["VAULT_TLS_SERVER_NAME"])
	assert.Equal(t, config["vaultClientCertFromSecret"], cm["VAULT_CLIENT_CERT"])
	assert.Equal(t, config["vaultClientCertKeyFromSecret"], cm["VAULT_CLIENT_KEY"])
	assert.Equal(t, config["vaultNamespace"], cm["VAULT_NAMESPACE"])
	assert.Equal(t, config["vaultCAVerify"], "false")
}

func TestVaultTokensKMSRegistered(t *testing.T) {
	t.Parallel()
	_, ok := kmsManager.providers[kmsTypeVaultTokens]
	assert.True(t, ok)
}
