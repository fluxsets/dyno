package dyno

import (
	"github.com/AdamSLevy/flagbind"
	"github.com/spf13/pflag"
	"log"
	"os"
)

type Option struct {
	ID       string `json:"id" flag:"id;;Server ID"`
	Conf     string `json:"conf" flag:"conf;./configs;config path, eg:--conf ./configs"`
	LogLevel string `json:"loglevel" flag:"loglevel;debug;default log level"`
	KWArgs   string `json:"kwargs" flag:"kwargs;;extern args, eg: --kwargs a=1,b=2"`
}

func BindOption() Option {
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
