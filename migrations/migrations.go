// Package migrations embeds the SQL schema of every service database so each
// service can apply its own migrations at startup.
package migrations

import "embed"

//go:embed all:auth all:catalog all:booking all:payment
var Files embed.FS
