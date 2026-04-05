// Package bundled embeds the default skill files and loader patches that ship
// with VibeMenu. The embedded FS mirrors the layout callers expect:
//
//	skills/<name>.md
//	loader-patches/<name>.txt
//
// Use [Extract] to unpack the embedded tree into a local directory.
package bundled

import "embed"

// SkillsFS holds all skill markdown files (skills/*.md).
//
//go:embed skills
var SkillsFS embed.FS

// PatchesFS holds all loader-patch text files (loader-patches/*.txt).
//
//go:embed loader-patches
var PatchesFS embed.FS
