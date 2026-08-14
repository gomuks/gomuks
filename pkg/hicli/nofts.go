//go:build !sqlite_fts5 && !fts5 && !libsqlite3

package hicli

var _ = missingSQLiteFTS5 + "hicli now requires -tags sqlite_fts5 to be set when compiling"
