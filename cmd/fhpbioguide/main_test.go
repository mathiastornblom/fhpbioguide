package main

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	"fhpbioguide/pkg/api/bioguide"

	"github.com/spf13/viper"
)

func loadCfg(t *testing.T, yaml string) {
	t.Helper()
	viper.Reset()
	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(strings.NewReader(yaml)); err != nil {
		t.Fatal(err)
	}
	viper.SetDefault("bio.id", 26)
	viper.SetDefault("bio.name", "FHP Riks")
	appLog = slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCreateCashAccounts(t *testing.T) {
	loadCfg(t, `
bio:
  URL: 'https://service.bioguiden.se'
  Username: 'fhpdist'
  Password: 'a'
  extraCashDistributors:
    - name: 'FHP Service'
      id: 423
      Username: 'fhpSeRV'
      Password: 'b'
`)
	acc := createCashAccounts(&bioguide.BioGuiden{Username: "fhpdist"})
	if len(acc) != 2 || acc[0].id != 26 || acc[0].name != "FHP Riks" {
		t.Fatalf("riks: %+v", acc)
	}
	if acc[1].id != 423 || acc[1].client.Username != "fhpSeRV" || acc[1].client.Password != "b" || acc[1].client.URL != "https://service.bioguiden.se" {
		t.Fatalf("service: %+v", acc[1])
	}
}

func TestCreateCashAccountsOnlyRiks(t *testing.T) {
	loadCfg(t, "bio:\n  Username: 'x'\n")
	if acc := createCashAccounts(&bioguide.BioGuiden{}); len(acc) != 1 || acc[0].id != 26 {
		t.Fatalf("%+v", acc)
	}
}

func TestCreateCashAccountsRejectsMissingPassword(t *testing.T) {
	loadCfg(t, "bio:\n  extraCashDistributors:\n    - name: S\n      id: 423\n      Username: u\n")
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for missing password")
		}
	}()
	createCashAccounts(&bioguide.BioGuiden{})
}
