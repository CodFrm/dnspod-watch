package pushcat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/codfrm/cago"
	"github.com/codfrm/cago/configs"
	"io"
	"net/http"
)

type Config struct {
	AccessToken []string `yaml:"accessToken"`
}

type Data struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

var defaultConfig *Config

func Pushcat() cago.FuncComponent {
	return func(ctx context.Context, cfg *configs.Config) error {
		defaultConfig = &Config{}
		if err := cfg.Scan("pushcat", defaultConfig); err != nil {
			return err
		}
		return nil
	}
}

func Send(ctx context.Context, title, msg string) error {
	url := "https://sct.icodef.com/openapi/v1/message/send"

	data := &Data{
		Title:   title,
		Content: msg,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	for _, token := range defaultConfig.AccessToken {
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		respData, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return errors.New(string(respData))
		}
	}
	return nil
}
