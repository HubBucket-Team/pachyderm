package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	vault "github.com/hashicorp/vault/api"

	"github.com/pachyderm/pachyderm/src/client"
	"github.com/pachyderm/pachyderm/src/client/auth"
)

const (
	vaultAddress = "http://127.0.0.1:8200"
	pachdAddress = "127.0.0.1:30650"
	pluginName   = "pachyderm"
)

func configurePlugin(t *testing.T, v *vault.Client) {

	c, err := client.NewFromAddress(pachdAddress)
	if err != nil {
		t.Errorf(err.Error())
	}
	resp, err := c.Authenticate(
		context.Background(),
		&auth.AuthenticateRequest{GitHubUsername: "admin", GitHubToken: "y"})

	if err != nil {
		t.Errorf(err.Error())
	}

	vl := v.Logical()
	config := make(map[string]interface{})
	config["admin_token"] = resp.PachToken
	config["pachd_address"] = pachdAddress
	_, err = vl.Write(
		fmt.Sprintf("/%v/config", pluginName),
		config,
	)

	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestConfig(t *testing.T) {
	vaultClientConfig := vault.DefaultConfig()
	vaultClientConfig.Address = vaultAddress
	v, err := vault.NewClient(vaultClientConfig)
	if err != nil {
		t.Errorf(err.Error())
	}
	v.SetToken("root")

	configurePlugin(t, v)

	vl := v.Logical()
	secret, err := vl.Read(
		fmt.Sprintf("/%v/config", pluginName),
	)
	fmt.Printf("config response: %v\n", secret)
	// We'll see an error if the admin token / pachd address are not set
	if err != nil {
		t.Errorf(err.Error())
	}

	if secret.Data["pachd_address"] != pachdAddress {
		t.Errorf("pachd_address configured incorrectly")
	}
	if secret.Data["ttl"] == "0s" {
		t.Errorf("ttl configured incorrectly")
	}

}

func loginHelper(t *testing.T) (*client.APIClient, *vault.Secret) {
	vaultClientConfig := vault.DefaultConfig()
	vaultClientConfig.Address = vaultAddress
	v, err := vault.NewClient(vaultClientConfig)
	if err != nil {
		t.Errorf(err.Error())
	}
	v.SetToken("root")

	configurePlugin(t, v)

	// Now hit login endpoint w invalid vault token, expect err
	params := make(map[string]interface{})
	params["username"] = "daffyduck"
	vl := v.Logical()
	secret, err := vl.Write(
		fmt.Sprintf("/%v/login", pluginName),
		params,
	)

	if err != nil {
		t.Errorf(err.Error())
	}

	pachToken, ok := secret.Auth.Metadata["user_token"]
	if !ok {
		t.Errorf("vault login response did not contain user token")
	}
	reportedPachdAddress, ok := secret.Auth.Metadata["pachd_address"]
	if !ok {
		t.Errorf("vault login response did not contain pachd address")
	}

	// Now do the actual test:
	// Try and list admins w a client w a valid pach token
	c, err := client.NewFromAddress(reportedPachdAddress)
	if err != nil {
		t.Errorf(err.Error())
	}
	c.SetAuthToken(pachToken)

	return c, secret
}

func TestLogin(t *testing.T) {
	// Negative control:
	//     Before we have a valid pach token, we should not
	// be able to list admins
	c, err := client.NewFromAddress(pachdAddress)
	if err != nil {
		t.Errorf(err.Error())
	}
	_, err = c.AuthAPIClient.GetAdmins(context.Background(), &auth.GetAdminsRequest{})
	if err == nil {
		t.Errorf("client could list admins before using auth token. this is likely a bug")
	}

	c, secret := loginHelper(t)

	_, err = c.AuthAPIClient.GetAdmins(c.Ctx(), &auth.GetAdminsRequest{})
	if err != nil {
		t.Errorf(err.Error())
	}

	fmt.Printf("sleeping for %vs\n", secret.LeaseDuration)
	time.Sleep(time.Duration(secret.LeaseDuration) * time.Second)
	// Just a bit extra to make sure we pass the expiry
	time.Sleep(time.Second)
	_, err = c.AuthAPIClient.GetAdmins(c.Ctx(), &auth.GetAdminsRequest{})
	if err == nil {
		t.Errorf("API call should fail, but token did not expire")
	}
}

func TestRenewBeforeTTLExpires(t *testing.T) {

}

func TestRevoke(t *testing.T) {
	// Do normal login
	// Use user token to connect
	// Issue revoke
	// Now renewal should fail ... but that token should still work? AFAICT
}
