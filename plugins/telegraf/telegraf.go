package telegraf

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/amonapp/amonagent/internal/util"
	"github.com/amonapp/amonagent/plugins"
	"github.com/mitchellh/mapstructure"
)

// Telegraf - XXX
type Telegraf struct {
	Config Config
}

// Description - XXX
func (t *Telegraf) Description() string {
	return "Collects data from Telegraf"
}

// Start - XXX
func (t *Telegraf) Start() error {
	return nil
}

// Stop - XXX
func (t *Telegraf) Stop() {
}

var sampleConfig = `
#   Available config options:
#
#     {"config": "/etc/opt/telegraf/telegraf.conf"}
#
#
# Config location: /etc/opt/amonagent/plugins-enabled/telegraf.conf
`

// SampleConfig - XXX
func (t *Telegraf) SampleConfig() string {
	return sampleConfig
}

// Config - XXX
type Config struct {
	Config string `mapstructure:"config"`
}

func (m Metric) String() string {
	s, _ := json.Marshal(m)
	return string(s)
}

// Metric - XXX
type Metric struct {
	Plugin string `json:"plugin"`
	Gauge  string `json:"gauge"`
	Value  string `json:"value"`
}

// ParsedLine - XXX
type ParsedLine struct {
	Elements []Metric
}

// SetConfigDefaults - XXX
func (t *Telegraf) SetConfigDefaults() error {
	configFile, err := plugins.UmarshalPluginConfig("telegraf")
	if err != nil {
		log.WithFields(log.Fields{"plugin": "telegraf", "error": err.Error()}).Error("Can't read config file")

		return err
	}
	var config Config
	e := mapstructure.Decode(configFile, &config)
	if e != nil {

		log.WithFields(log.Fields{"plugin": "telegraf", "error": e.Error()}).Error("Can't decode config file")

		return e
	}

	t.Config = config

	return nil
}

// ParseLine - XXX
func (t *Telegraf) ParseLine(s string) (ParsedLine, error) {
	line := ParsedLine{}
	// split by space
	space := func(c rune) bool {
		return c == ' '
	}

	// split by =,
	eq := func(c rune) bool {
		return c == '='
	}

	// split by ,
	comma := func(c rune) bool {
		return c == ','
	}

	//split metric name by _
	underscore := func(c rune) bool {
		return c == '_'
	}

	measurementLine := strings.FieldsFunc(s, space)
	// line := ParsedLine{}
	// skip non-essential information like * Plugin: name
	if len(measurementLine) > 0 {

		lineStarter := measurementLine[0]
		// > ping,url=www.google.com average_response_ms=2.596,packets_received=1i 1454321712994367057
		if lineStarter == ">" {

			if len(measurementLine) == 4 {
				// ping,url=www.google.com
				pluginMeta := strings.FieldsFunc(measurementLine[1], comma)

				if len(pluginMeta) > 1 {

					validTags := []string{}
					// TODO - Extend to a list in the future
					// Range over tags and ignore the host values
					for _, v := range pluginMeta[1:] {

						startsWith := strings.HasPrefix(v, "host")
						if startsWith == false {
							validTags = append(validTags, v)
						}

					}
					chartName := strings.Join(validTags, "|") // url=google.com
					chartName = strings.Replace(chartName, ".", "", -1)
					chartName = strings.Replace(chartName, "=", ":", -1)

					metricValue := strings.FieldsFunc(measurementLine[2], comma)
					for _, v := range metricValue {
						m := Metric{}
						// inodes_used=0i
						// total=0i

						metric := strings.FieldsFunc(v, eq)
						if len(metric) == 2 {
							var value string
							toFloat, err := strconv.ParseFloat(metric[1], 64)
							if err != nil {
								value = strings.Replace(metric[1], "i", "", -1)
							} else {
								value = strconv.FormatFloat(toFloat, 'f', -1, 64)
							}

							splitOnUnderscore := strings.FieldsFunc(metric[0], underscore)

							var cleanName string

							if len(splitOnUnderscore) > 2 {
								cleanName = strings.Join(splitOnUnderscore[0:], ".")
							} else {

								cleanName = strings.Join(splitOnUnderscore[:], ".")
							}

							m.Plugin = "telegraf." + pluginMeta[0] // ping
							m.Gauge = pluginMeta[0] + chartName + "_" + cleanName
							m.Value = value

							line.Elements = append(line.Elements, m)

						}

					}

				}

			}
		}

	}

	return line, nil

}

func contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

// Collect - XXX
func (t *Telegraf) Collect() (interface{}, error) {
	t.SetConfigDefaults()

	CommandString := fmt.Sprintf("/usr/bin/telegraf -test -config %s", t.Config.Config)
	var command = util.Command{Command: CommandString}

	Commandresult := util.ExecWithExitCode(command)

	plugins := make(map[string]interface{})
	lines := strings.Split(Commandresult.Output, "\n")
	var result []Metric
	for _, line := range lines {
		metrics, _ := t.ParseLine(line)

		if len(metrics.Elements) > 0 {
			for _, m := range metrics.Elements {
				if len(m.Gauge) > 0 {
					result = append(result, m)
				}
			}
		}

	}
	// Filter unique plugins
	AllPlugins := []string{}
	for _, r := range result {
		if !contains(AllPlugins, r.Plugin) {
			AllPlugins = append(AllPlugins, r.Plugin)
		}
	}
	for _, p := range AllPlugins {
		plugins[p] = make(map[string]interface{})
		GaugesWrapper := make(map[string]map[string]string)
		gauges := make(map[string]string)
		for _, r := range result {

			if r.Plugin == p {
				gauges[r.Gauge] = r.Value
			}

		}

		GaugesWrapper["gauges"] = gauges

		plugins[p] = GaugesWrapper

	}

	return plugins, nil
}

func init() {
	plugins.Add("telegraf", func() plugins.Plugin {
		return &Telegraf{}
	})
}
