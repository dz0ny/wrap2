package main

// Cron holds data about process to be executed
type Cron struct {
	Command  `toml:"job"`
	Schedule string `toml:"schedule, omitempty"`
}
