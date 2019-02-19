package servermanager

type Account struct {
	Name            string `yaml:"name"`
	Group           Group  `yaml:"group"`
	PasswordMD5Hash string `yaml:"password"`
}

type Group string
