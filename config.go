package dyno

import (
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
	"log"
)

type Config interface {
	ConfigGetter
	ConfigUnmarshaler
	Sub(key string) Config
	Merge(data map[string]interface{})
}

type ConfigGetter interface {
	Get(key string) any
	GetInt(key string) int
	GetBool(key string) bool
	GetString(key string) string
}

type ConfigUnmarshaler interface {
	Unmarshal(v any) error
}

type viperConfig struct {
	v *viper.Viper
}

func (vc *viperConfig) Merge(data map[string]interface{}) {
	for k, v := range data {
		vc.v.Set(k, v)
	}
}

func (vc *viperConfig) Get(key string) any {
	return vc.v.Get(key)
}

func (vc *viperConfig) GetInt(key string) int {
	return vc.v.GetInt(key)
}

func (vc *viperConfig) GetBool(key string) bool {
	return vc.v.GetBool(key)
}

func (vc *viperConfig) GetString(key string) string {
	return vc.v.GetString(key)
}

func (vc *viperConfig) Unmarshal(v any) error {
	return vc.v.Unmarshal(v, func(config *mapstructure.DecoderConfig) {
		config.TagName = "json"
	})
}

func (vc *viperConfig) Sub(key string) Config {
	sub := vc.v.Sub(key)
	return &viperConfig{
		v: sub,
	}
}

var _ Config = new(viperConfig)

func newConfig(paths []string, exts []string) Config {
	v := viper.New()
	for _, ext := range exts {
		v.SetConfigType(ext)
	}
	for _, path := range paths {
		v.AddConfigPath(path)
	}

	if err := v.ReadInConfig(); err != nil {
		log.Fatal(err)
	}
	return &viperConfig{
		v: v,
	}
}
