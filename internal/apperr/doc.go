// Package apperr defines shared error sentinels for the trident application.
// It is a leaf package with no internal imports, allowing any package
// (including low-level infrastructure like doh) to use the sentinels
// without creating import cycles.
package apperr
