/*
Copyright © 2022 Runar Kristoffersen

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/mattn/go-isatty"
	"github.com/runar-rkmedia/go-common/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stoewer/go-strcase"
)

var (
	cfgFile string
	l       logger.AppLogger = logger.GetLogger("")
	CLI     config
)

type secret string

func (u secret) MarshalJSON() ([]byte, error) {
	return []byte(`""`), nil
}
func (u secret) MarshalTOML() ([]byte, error) {
	return []byte(`""`), nil
}

type config struct {
	URI               string   `help:"Endpoint for skiver" env:"SKIVER_URI" short:"u" json:"uri"`
	Project           string   `help:"Project-id/ShortName" short:"p" env:"SKIVER_PROJECT" json:"project"`
	Token             secret   `help:"Token used for authentication" short:"t" env:"SKIVER_TOKEN" json:"token"`
	Locale            string   `help:"Locale to use" env:"SKIVER_LOCALE" short:"l" json:"locale"`
	WithPrettier      bool     `help:"Where available, will attempt to run prettier, or prettier_d if available" json:"with_prettier"`
	PrettierPath      string   `help:"Path-override for prettier" default:"prettier" json:"prettier_path"`
	PrettierDSlimPath string   `help:"Path-override for prettier_d_slim, which should be faster than regular prettier" default:"prettier_d_slim" json:"prettier_d_slim_path"`
	IgnoreFilter      []string `help:"Ignore-filter for files" json:"ignore_filter"`
	NoColor           bool     `help:"If set, disables color for printing of output. Does not affect logging." json:"no_color"`
	HighlightStyle    string   `help:"Highlighting-style to use. See https://github.com/alecthomas/chroma/tree/master/styles for valid styles" json:"highlight_style"`

	Import struct {
		Source string `help:"Source-file for import" arg:"" env:"SKIVER_IMPORT_SOURCE" json:"source"`
	} `help:"Import from file" cmd:"" json:"import"`
	Generate struct {
		Path   string `help:"Ouput file to write to" type:"path" env:"SKIVER_GENERATE_PATH" json:"path"`
		Format string `help:"Generate files from export. Common formats are: i18n,typescript." json:"format" required:"true"`
	} `help:"Generate files from project etc." cmd:"" json:"generate"`
	Unused struct {
		Source string `help:"Source-file to check-against. If ommitted, the upstream project is used as source" json:"source"`
		Dir    string `help:"Directory for source-code" type:"existingdir" arg:"" required:"" json:"dir"`
	} `help:"Find unused translation-keys" cmd:"" json:"unused"`

	Inject struct {
		DryRun    bool   `help:"Enable dry-run" json:"dry_run"`
		OnReplace string `help:"Command to run on file after replacement, like prettier" json:"on_replace"`
		Dir       string `help:"Directory for source-code" type:"existingdir" arg:"" json:"dir"`
	} `help:"Inject helper-comments into source-files" cmd:"" json:"inject"`
	Config struct {
		Format string `enum:"json,yaml,toml" default:"toml" json:"format"`
	} `help:"Configuration" cmd:"" json:"config"`
	LogFormat string `help:"Format to log as" default:"human" enum:"json,human" json:"log_format"`
	LogLevel  string `help:"Level for logging." default:"info" enum:"trace,debug,info,warn,error,panic" json:"log_level"`
}

var (
	configFile string
	_api       *Api

	// These are added at build...
	version       string
	date          string
	buildDate     time.Time
	builtBy       string
	commit        string
	isInteractive bool
)

func init() {
	if date != "" {
		t, err := time.Parse("2006-01-02T15:04:05Z", date)
		if err != nil {
			panic(fmt.Errorf("Failed to parse build-date: %w", err))
		}
		buildDate = t
	}
	if isatty.IsTerminal(os.Stdout.Fd()) {
		isInteractive = true
	} else if isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		isInteractive = true
	} else {
		isInteractive = false
	}
}

func requireApi(withAuthentciation bool) *Api {
	if _api == nil {
		if CLI.URI == "" {
			l.Fatal().Msg("URI is required")
		}
		if withAuthentciation && CLI.Token == "" {
			l.Fatal().Msg("Token is required")
		}

		a := NewAPI(l, CLI.URI)
		_api = &a

		_api.SetToken(string(CLI.Token))
	}
	if _api == nil {
		l.Fatal().Msg("Api is not initialized")
	}
	return _api
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "Skiver-CLI",
	Short: "Interactions with skiver, a developer-focused translation-service",

	Version: version,

	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) {

	// },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/skiver/skiver-cli.yaml)")

	s := reflect.TypeOf(CLI)
	for _, v := range []string{"HighlightStyle", "NoColor", "Project", "WithPrettier", "PrettierPath", "PrettierDSlimPath", "LogFormat", "LogLevel", "URI", "Locale", "Token", "IgnoreFilter"} {
		mustSetVar(s, v, rootCmd, "")
	}

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	godotenv.Load()
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".cli" (without extension).
		viper.SetConfigName("skiver-cli")
		viper.AddConfigPath(path.Join(home, "skiver"))
		viper.AddConfigPath(path.Join(home, ".config", "skiver"))
		viper.AddConfigPath(".")
		viper.SetEnvPrefix("skiver")
	}
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		configFile = viper.ConfigFileUsed()
	} else {

		l = logger.InitLogger(logger.LogConfig{
			Format:     "human",
			Level:      "debug",
			WithCaller: true,
		})
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			l.Fatal().Err(err).Msg("Config-file not found. You can create one with `skiver config new`")
		}
		l.Fatal().Err(err).Str("config-file", viper.ConfigFileUsed()).Msg("Failed to read config-file")
	}

	viper.AutomaticEnv() // read in environment variables that match
	err := (viper.Unmarshal(&CLI))
	if err != nil {
		panic(err)
	}

	l = logger.InitLogger(logger.LogConfig{
		Format:     CLI.LogFormat,
		Level:      CLI.LogLevel,
		WithCaller: CLI.LogLevel == "debug",
	})

}

func mustSetVar(t reflect.Type, name string, cmd *cobra.Command, subkey string) {
	err := setVar(t, name, cmd, subkey)
	if err != nil {
		panic(err)
	}
}
func setVar(t reflect.Type, name string, cmd *cobra.Command, subkey string) error {

	field, ok := t.FieldByName(name)
	if !ok {
		var fieldNames []string
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			fieldNames = append(fieldNames, f.Name)

		}
		return fmt.Errorf("[%s] field '%s' not found within struct. Available names: %v", cmd.Name(), name, fieldNames)
	}

	cfgName := field.Tag.Get("cfg")
	if cfgName == "-" {
		return nil
	}
	if cfgName == "" {
		cfgName = strcase.KebabCase(name)

		// return fmt.Errorf("field %s has no name %#v", field.Name, field)
	}
	short := field.Tag.Get("short")
	mapstructure := field.Tag.Get("mapstructure")
	if mapstructure == "" {
		mapstructure = field.Tag.Get("json")
	}
	if mapstructure == "" {
		return fmt.Errorf("Field '%s' of '%s' does not have a 'mapstructure'- or 'json'-tag, and should have one for consistency", field.Name, t.Name())
		// This works, but does not reflect the raw fieldname, so it causes confusion

		// mapstructure = strcase.SnakeCase(name)
	}
	defaultStr := field.Tag.Get("default")
	desc := field.Tag.Get("help")
	kind := field.Type.Name()
	if kind == "" {
		kind = field.Type.String()
	}
	switch kind {
	case "bool":
		defaultValue := defaultStr == "true"
		cmd.PersistentFlags().BoolP(cfgName, short, defaultValue, desc)
	case "string", "secret":
		cmd.PersistentFlags().StringP(cfgName, short, defaultStr, desc)

	case "int":
		defaultInt := 0
		if defaultStr != "" {
			n, err := strconv.ParseInt(defaultStr, 10, 64)
			if err != nil {
				panic(fmt.Sprintf("failed to convert default-tag (%s) on config-field %s", defaultStr, field.Name))
			}
			defaultInt = int(n)
		}
		cmd.PersistentFlags().IntP(cfgName, short, defaultInt, desc)
	case "[]string":
		var defaultStrings []string
		if defaultStr != "" {
			defaultStrings = strings.Split(defaultStr, ",")
		}
		cmd.PersistentFlags().StringSliceP(cfgName, short, defaultStrings, desc)
	case "[]int":
		var defaultInts []int
		if defaultStr != "" {
			split := strings.Split(defaultStr, ",")
			for i := 0; i < len(split); i++ {
				n, err := strconv.ParseInt(split[i], 10, 64)
				if err != nil {
					panic(fmt.Sprintf("failed to convert default-tag (%s) on config-field %s", defaultStr, field.Name))
				}
				defaultInts = append(defaultInts, int(n))
			}
		}
		cmd.PersistentFlags().IntSliceP(cfgName, short, defaultInts, desc)
	case "interface {}":
		cmd.PersistentFlags().StringP(cfgName, short, defaultStr, desc)
	case "map[string]string":
		var defaultStrings map[string]string
		if defaultStr != "" {
			split := strings.Split(defaultStr, ";")
			for i := 0; i < len(split); i++ {
				kv := strings.Split(defaultStr, "=")
				defaultStrings[kv[0]] = kv[1]
			}
		}
		cmd.PersistentFlags().StringToStringP(cfgName, short, defaultStrings, desc)
	case "map[string]interface {}":
		return nil
	default:
		panic(fmt.Sprintf("no handler for %s, %s", field.Name, kind))
	}
	viper.BindPFlag(subkey+mapstructure, cmd.PersistentFlags().Lookup(cfgName))
	// fmt.Printf("\n'%s%s' '%s' ", subkey, mapstructure, cfgName)
	alias := strings.ToLower(name)
	if mapstructure != alias {
		// fmt.Println("\n   >>>> alias", alias, subkey+mapstructure)
		viper.RegisterAlias(subkey+alias, subkey+mapstructure)
	}
	return nil
}
