package config

import (
	"github.com/AnyISalIn/sshwrapper/handlers"
)

type Config struct {
	Routes      []Route `yaml:"routes"`
	Users       []User  `yaml:"users"`
	HostKeyFile *string `yaml:"hostKeyFile,omitempty"`
}

type Route struct {
	URI           string        `yaml:"uri"` // URI similar http path eg: /bastion/:id
	Handler       HandlerConfig `yaml:"handler"`
	AllowUsers    []string      `yaml:"allow,omitempty"`
	AllowAllUsers bool          `yaml:"allowAllUsers,omitempty"`
}

func (r *Route) Allow(user User) bool {
	if r.AllowAllUsers {
		return true
	}
	for _, u := range r.AllowUsers {
		if u == user.Name {
			return true
		}
	}
	return false
}

type HandlerConfig struct {
	Command *handlers.CommandConfig `json:"command,omitempty"`
}

type User struct {
	Name          string  `yaml:"name"`
	Password      *string `yaml:"password,omitempty"`
	PublicKey     *string `yaml:"publicKey,omitempty"`
	PublicKeyPath *string `yaml:"publicKeyPath,omitempty"`
}
