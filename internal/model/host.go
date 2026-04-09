package model

type Host struct {
	Alias        string
	Hostname     string
	User         string
	Port         string
	IdentityFile string
	Group        string
}

type Group struct {
	Name  string
	Hosts []Host
}
