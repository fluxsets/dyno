package option

import (
	"encoding/json"
	"github.com/AdamSLevy/flagbind"
	"github.com/spf13/pflag"
	"log"
	"os"
	"strings"
	"time"
)

type Option struct {
	ID              string        `json:"id" flag:"id;;Server ID"`
	ConfigDir       string        `json:"config_dir" flag:"conf;./configs;config path, eg:--config_dir ./configs"`
	ConfigType      string        `json:"config_type" flag:"config_type;config file type, eg:--config_type yaml"`
	Config          string        `json:"config" flag:"config;config file, eg: --config ./configs/config.yaml"`
	LogLevel        string        `json:"loglevel" flag:"loglevel;debug;default log level"`
	KWArgs          string        `json:"kwargs" flag:"kwargs;;extern args, eg: --kwargs a=1,b=2"`
	Version         string        `json:"version"`
	Name            string        `json:"name"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
}

func (o *Option) String() string {
	bs, _ := json.Marshal(o)
	return string(bs)

}
func (o *Option) EnsureDefaults() {
	if o.ID == "" {
		o.ID, _ = os.Hostname()
	}
	o.ShutdownTimeout = 5 * time.Second
}

func (o *Option) KWArgsAsMap() map[string]any {
	kwargs := map[string]any{}
	args := strings.Split(o.KWArgs, ",")
	for _, s := range args {
		kv := strings.Split(s, "=")
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		kwargs[key] = value
	}
	return kwargs
}

func FromFlags() Option {
	fs := pflag.NewFlagSet("", pflag.ExitOnError)
	option := Option{}
	if err := flagbind.Bind(fs, &option); err != nil {
		log.Fatalln(err)
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
	return option
}
