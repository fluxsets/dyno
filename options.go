package orbit

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
	Conf            string        `json:"conf" flag:"conf;./configs;config path, eg:--conf ./configs"`
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
func (o *Option) ensureDefaults() {
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

func OptionFromFlags() Option {
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
