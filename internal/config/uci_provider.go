package config

import (
	"fmt"
	"github.com/BoRuDar/configuration/v4"
	"kiezbox/internal/utils"
	"reflect"
)

const (
	UciProviderName = `UciProvider`
	UciProviderTag  = `uci`
)

// NewUciProvider creates new provider which reads values from uci config
func NewUciProvider(uciBase string) *UciProvider {
	return &UciProvider{uciBase: uciBase}
}

type UciProvider struct {
	uciBase string
}

func (fp *UciProvider) Name() string {
	return UciProviderName
}

func (fp *UciProvider) Tag() string {
	return UciProviderTag
}

func (fp *UciProvider) Init(_ any) error {
	//TODO: consider checking if uci is available
	// but this breaks testing currently
	// return utils.UciCheck()
	return nil
}

func (fp *UciProvider) Provide(field reflect.StructField, v reflect.Value) error {
	uci_path := field.Tag.Get(UciProviderTag)
	if len(uci_path) == 0 {
		// field doesn't have a proper tag
		return fmt.Errorf("%s: uci_path is empty", UciProviderName)
	}
	valStr, err := utils.UciGet(fp.uciBase + uci_path)
	if err != nil {
		return fmt.Errorf("%s: UciGet returns empty value", UciProviderName)
	}

	return configuration.SetField(field, v, valStr)
}
