package credentials

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/gfx-labs/scram"

	"gfx.cafe/gfx/pggat/lib/auth"
)

// exchange runs a SASL exchange the way the bouncer frontend and backend drive it.
func exchange(t *testing.T, client auth.SASLClient, server auth.SASLServer) error {
	t.Helper()
	mechanism, encoder, err := client.EncodeSASL(server.SupportedSASLMechanisms())
	if err != nil {
		return err
	}
	verifier, err := server.VerifySASL(mechanism)
	if err != nil {
		return err
	}
	msg, err := encoder.Step(nil)
	if err != nil {
		return err
	}
	for {
		resp, err := verifier.Step(msg)
		if err != nil {
			return err
		}
		if verifier.Done() {
			if _, err := encoder.Step(resp); err != nil {
				return err
			}
			if !encoder.Authenticated() {
				return fmt.Errorf("client did not authenticate server")
			}
			return nil
		}
		if msg, err = encoder.Step(resp); err != nil {
			return err
		}
	}
}

func scramSecret(t *testing.T, password string) string {
	t.Helper()
	info, err := scram.NewKeyInfo(sha256.New, 4096)
	if err != nil {
		t.Fatal(err)
	}
	keys, err := scram.DeriveServerKeys(password, info)
	if err != nil {
		t.Fatal(err)
	}
	b64 := base64.StdEncoding.EncodeToString
	return fmt.Sprintf("SCRAM-SHA-256$%d:%s$%s:%s", info.Iters, b64(info.Salt), b64(keys.StoredKey), b64(keys.ServerKey))
}

func TestCleartextSASL(t *testing.T) {
	good := Cleartext{Username: "u", Password: "pw"}
	if err := exchange(t, good, good); err != nil {
		t.Fatalf("matching password: %v", err)
	}
	if err := exchange(t, Cleartext{Username: "u", Password: "wrong"}, good); err == nil {
		t.Fatal("wrong password authenticated")
	}
}

func TestScramPassthrough(t *testing.T) {
	secret := scramSecret(t, "pw")
	proxy, err := ScramFromString(secret)
	if err != nil {
		t.Fatal(err)
	}
	upstream, err := ScramFromString(secret)
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := proxy.EncodeSASL([]auth.SASLMechanism{auth.ScramSHA256}); err == nil {
		t.Fatal("EncodeSASL succeeded before any client logged in")
	}

	if err := exchange(t, Cleartext{Password: "wrong"}, proxy); err == nil {
		t.Fatal("wrong password authenticated")
	}
	if _, _, err := proxy.EncodeSASL([]auth.SASLMechanism{auth.ScramSHA256}); err == nil {
		t.Fatal("failed login recorded client keys")
	}

	if err := exchange(t, Cleartext{Password: "pw"}, proxy); err != nil {
		t.Fatalf("client login: %v", err)
	}
	// The proxy reuses the recovered keys to log in upstream without the password.
	if err := exchange(t, proxy, upstream); err != nil {
		t.Fatalf("upstream login with recovered keys: %v", err)
	}
}
