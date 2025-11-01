package conf

import (
	"encoding/json"
	"os"
)

type ConfList struct {
	CertSourceDir string    `json:"certSourceDir"`
	List          []*Config `json:"confList"`
}
type Config struct {
	Domain       string    `json:"domain"`
	AccessName   string    `json:"access_name"`
	DNS          string    `json:"dns"`
	AccessKey    string    `json:"access_key"`
	AccessSecret string    `json:"access_secret"`
	Name         string    `json:"name"`
	Targets      []*Target `json:"targets"`
	Status       int       `json:"status"`
	CreatedAt    string    `json:"created_at"`
	UpdatedAt    string    `json:"updated_at"`
}

type Target struct {
	TargetMod  int    `json:"target_mod"`
	TargetName string `json:"target_name"`
	TargetUser string `json:"target_user"`
	TargetKey  string `json:"target_sshKey"`
	TargetIP   string `json:"target_ip"`
	TargetPort string `json:"target_port"`
	TargetDir  string `json:"target_dir"`
	ReloadCmd  string `json:"reload_cmd"`
	Status     int    `json:"status"`
	Remark     string `json:"remark"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

func Load(file string, c *ConfList) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	return json.Unmarshal(content, c)
}
