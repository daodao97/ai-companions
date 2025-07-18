package conf

import (
	"fmt"
	"log/slog"

	"github.com/daodao97/xgo/xapp"
	"github.com/daodao97/xgo/xdb"

	"github.com/daodao97/xgo/xlog"
)

type TTSConfig struct {
	Name     string `yaml:"name"`
	Provider string `yaml:"provider"`
	GroupId  string `yaml:"group_id"`
	ApiKey   string `yaml:"api_key"`
	ApiUrl   string `yaml:"api_url"`
	Model    string `yaml:"model"`
	Voice    string `yaml:"voice"`
	Speed    string `yaml:"speed" default:"1.0"`
	Volume   string `yaml:"volume" default:"1.0"`
	Pitch    string `yaml:"pitch" default:"0"`
	Format   string `yaml:"format" default:"mp3"`
}

type STTConfig struct {
	Name     string `yaml:"name"`
	Provider string `yaml:"provider"`
	ApiKey   string `yaml:"api_key"`
	ApiUrl   string `yaml:"api_url"`
	Model    string `yaml:"model"`
	Voice    string `yaml:"voice"`
}

type LLMConfig struct {
	Name      string `yaml:"name"`
	Provider  string `yaml:"provider"`
	ApiKey    string `yaml:"api_key"`
	ApiUrl    string `yaml:"api_url"`
	Model     string `yaml:"model"`
	Voice     string `yaml:"voice"`
	MaxTokens int    `yaml:"max_tokens"`
}

type config struct {
	JwtSecret string       `yaml:"jwt_secret"`
	AdminPath string       `yaml:"admin_path"`
	Database  []xdb.Config `yaml:"database" envPrefix:"DATABASE"`
	TTS       []*TTSConfig `yaml:"tts"`
	STT       []*STTConfig `yaml:"stt"`
	LLM       []*LLMConfig `yaml:"llm"`
}

func (c *config) GetTTS(name string) *TTSConfig {
	for _, tts := range c.TTS {
		if tts.Name == name {
			return tts
		}
	}
	return nil
}

func (c *config) GetSTT(name string) *STTConfig {
	for _, stt := range c.STT {
		if stt.Name == name {
			return stt
		}
	}
	return nil
}

func (c *config) GetLLM(name string) *LLMConfig {
	for _, llm := range c.LLM {
		if llm.Name == name {
			return llm
		}
	}
	return nil
}

func (c *config) Print() {
	xlog.Debug("load config", slog.Any("config", fmt.Sprintf("%+v", c)))
}

var _c *config

func Get() *config {
	return _c
}

func InitConf() error {
	_c = &config{
		AdminPath: "/_",
	}

	if err := xapp.InitConf(_c); err != nil {
		return err
	}

	_c.Print()

	return nil
}
